package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
	"github.com/c7d5a6/c7d5a6l/internal/telegram"
)

const (
	notifyJaedong = "https://liquipedia.net/starcraft/Jaedong"
	notifyFlash   = "https://liquipedia.net/starcraft/Flash"
)

type fakeGroupBot struct {
	sent               []string
	failNext           int
	members            map[int64]telegram.ChatMember
	getChatMemberCalls int
}

func (f *fakeGroupBot) Configured() bool { return true }

func (f *fakeGroupBot) SendGroup(_ context.Context, text string) error {
	if f.failNext > 0 {
		f.failNext--
		return errors.New("send failed")
	}
	f.sent = append(f.sent, text)
	return nil
}

func (f *fakeGroupBot) GetChatMember(_ context.Context, userID int64) (*telegram.ChatMember, error) {
	f.getChatMemberCalls++
	if f.members == nil {
		return &telegram.ChatMember{Status: "left"}, nil
	}
	m, ok := f.members[userID]
	if !ok {
		return &telegram.ChatMember{Status: "left", User: telegram.User{ID: userID}}, nil
	}
	return &m, nil
}

type notifyFix struct {
	ctx     context.Context
	notify  *service.TelegramNotify
	bot     *fakeGroupBot
	fantasy *service.Fantasy
	tours   *service.Tournament
	users   *repository.User
	league  model.FantasyLeague
	page    model.TournamentPage
	jdID    int64
	flashID int64
}

func setupNotify(t *testing.T, firstStart time.Time, jaedongWinner bool) *notifyFix {
	t.Helper()
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	playerRepo := repository.NewPlayer(sqlDB)
	tourRepo := repository.NewTournament(sqlDB)
	userRepo := repository.NewUser(sqlDB)
	tours := service.NewTournament(sqlDB, tourRepo, playerRepo, nil, stubPlayerFetcher{
		notifyJaedong: {Name: str("Jaedong"), PreferredRace: str("zerg"), IDs: []string{}},
		notifyFlash:   {Name: str("Flash"), PreferredRace: str("terran"), IDs: []string{}},
	}, nil)
	dt := firstStart.UTC().Format(time.RFC3339)
	page := model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/ASL/notify",
		Name: str("Notify Cup"),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str(notifyJaedong), Race: str("zerg")},
			{Name: str("Flash"), Link: str(notifyFlash), Race: str("terran")},
		},
		Groups: []model.TournamentGroup{{
			Name: "Group A", Phase: "Round of 24",
			Players: []model.Participant{
				{Name: str("Jaedong"), Link: str(notifyJaedong), Race: str("zerg"), IsWinner: jaedongWinner},
				{Name: str("Flash"), Link: str(notifyFlash), Race: str("terran")},
			},
		}},
		Results: []model.Result{{
			Played: false, Phase: "Round of 24", Round: "Group A", Order: 1,
			DateTime:     &dt,
			ParticipantA: &model.Participant{Name: str("Jaedong"), Link: str(notifyJaedong), Race: str("zerg")},
			ParticipantB: &model.Participant{Name: str("Flash"), Link: str(notifyFlash), Race: str("terran")},
		}},
	}
	if _, _, _, err := tours.Save(ctx, page); err != nil {
		t.Fatal(err)
	}
	var tournamentID int64
	if err := sqlDB.QueryRowContext(ctx, `SELECT id FROM tournament WHERE link = ?`, page.Link).Scan(&tournamentID); err != nil {
		t.Fatal(err)
	}

	fantasyRepo := repository.NewFantasy(sqlDB)
	fantasy := service.NewFantasy(sqlDB, fantasyRepo, tourRepo, nil)
	league, err := fantasy.CreateOrSeed(ctx, tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fantasy.StartLeague(ctx, league.ID); err != nil {
		t.Fatal(err)
	}

	players, err := fantasy.ListPlayers(ctx, league.ID, repository.PlayerSortCost)
	if err != nil || len(players) != 2 {
		t.Fatalf("players=%d err=%v", len(players), err)
	}
	var jdID, flashID int64
	for _, p := range players {
		if p.Name != nil && *p.Name == "Jaedong" {
			jdID = p.ID
		}
		if p.Name != nil && *p.Name == "Flash" {
			flashID = p.ID
		}
	}

	bot := &fakeGroupBot{members: map[int64]telegram.ChatMember{}}
	notify := service.NewTelegramNotify(sqlDB, fantasy, userRepo, repository.NewTelegramNotice(), bot, 15*time.Minute)
	return &notifyFix{
		ctx: ctx, notify: notify, bot: bot, fantasy: fantasy, tours: tours, users: userRepo,
		league: league, page: page, jdID: jdID, flashID: flashID,
	}
}

func (f *notifyFix) addUser(t *testing.T, alias string, tgID *int64, username *string) int64 {
	t.Helper()
	id, err := f.users.Insert(f.ctx, f.users.DB(), model.User{
		Alias:            alias,
		FirstName:        alias,
		Role:             model.RoleUser,
		TelegramID:       tgID,
		TelegramUsername: username,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *notifyFix) roster(t *testing.T, userID int64, playerIDs ...int64) {
	t.Helper()
	if _, err := f.fantasy.UpsertTeam(f.ctx, service.UpsertTeamParams{
		LeagueID:         f.league.ID,
		UserID:           userID,
		FantasyPlayerIDs: playerIDs,
	}); err != nil {
		t.Fatal(err)
	}
}

func (f *notifyFix) savePlayed(t *testing.T, winnerJaedong bool) {
	t.Helper()
	f.page.Results[0].Played = true
	f.page.Results[0].ScoreA = intPtr(2)
	f.page.Results[0].ScoreB = intPtr(0)
	f.page.Groups[0].Players[0].IsWinner = winnerJaedong
	if _, _, _, err := f.tours.Save(f.ctx, f.page); err != nil {
		t.Fatal(err)
	}
}

func TestOperatorMention(t *testing.T) {
	t.Parallel()
	id := int64(42)
	user := "raynor"
	cases := []struct {
		name     string
		alias    string
		id       *int64
		username *string
		inGroup  bool
		off      bool
		want     string
	}{
		{name: "in group tags", alias: "Jim", id: &id, username: &user, inGroup: true, want: "@raynor"},
		{name: "in group no username", alias: "Jim", id: &id, inGroup: true, want: "Jim"},
		{name: "id not in group", alias: "Jim", id: &id, username: &user, inGroup: false, want: "Jim"},
		{name: "no id still at-username", alias: "Ghost", username: &user, want: "@raynor"},
		{name: "alias only", alias: "Pilot", want: "Pilot"},
		{name: "disabled uses alias not username", alias: "Jim", id: &id, username: &user, inGroup: true, off: true, want: "Jim"},
		{name: "disabled strips leading at", alias: "@Pilot", username: &user, inGroup: true, off: true, want: "Pilot"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := service.OperatorMention(tc.alias, tc.id, tc.username, tc.inGroup, !tc.off)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestTelegramNotify_prematchWindow(t *testing.T) {
	first := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	f := setupNotify(t, first, false)
	uid := f.addUser(t, "Commander", nil, str("cmdr"))
	f.roster(t, uid, f.jdID)

	f.notify.Tick(f.ctx, first.Add(-16*time.Minute))
	if len(f.bot.sent) != 0 {
		t.Fatalf("too early sent=%v", f.bot.sent)
	}

	f.notify.Tick(f.ctx, first.Add(-15*time.Minute))
	if len(f.bot.sent) != 1 {
		t.Fatalf("at lead sent=%v", f.bot.sent)
	}
	msg := f.bot.sent[0]
	if !strings.Contains(msg, "Сегодня в Round of 24 играют") {
		t.Fatalf("phase line: %s", msg)
	}
	if !strings.Contains(msg, "Jaedong") || !strings.Contains(msg, "Flash") {
		t.Fatalf("players: %s", msg)
	}
	if !strings.Contains(msg, "Удачи командам @cmdr!") {
		t.Fatalf("operators: %s", msg)
	}

	f.notify.Tick(f.ctx, first.Add(-10*time.Minute))
	if len(f.bot.sent) != 1 {
		t.Fatalf("dedup sent=%v", f.bot.sent)
	}
}

func TestTelegramNotify_prematchAfterStartSameDay(t *testing.T) {
	first := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	f := setupNotify(t, first, false)

	f.notify.Tick(f.ctx, first.Add(2*time.Hour))
	if len(f.bot.sent) != 1 {
		t.Fatalf("after start sent=%v", f.bot.sent)
	}
	if strings.Contains(f.bot.sent[0], "Удачи командам") {
		t.Fatalf("no operators expected: %s", f.bot.sent[0])
	}

	f.notify.Tick(f.ctx, first.Add(24*time.Hour))
	if len(f.bot.sent) != 1 {
		t.Fatalf("next day should not send again=%v", f.bot.sent)
	}
}

func TestTelegramNotify_dayDoneWaitsForWinners(t *testing.T) {
	first := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	f := setupNotify(t, first, false)
	uid := f.addUser(t, "Jim", int64Ptr(9001), str("raynor"))
	f.bot.members[9001] = telegram.ChatMember{Status: "member", User: telegram.User{ID: 9001, Username: "raynor"}}
	f.roster(t, uid, f.jdID)

	now := first.Add(3 * time.Hour)
	f.notify.Tick(f.ctx, now)
	if len(f.bot.sent) != 1 || !strings.Contains(f.bot.sent[0], "Сегодня") {
		t.Fatalf("expected prematch first: %v", f.bot.sent)
	}

	f.savePlayed(t, false)
	f.notify.Tick(f.ctx, now)
	if len(f.bot.sent) != 2 {
		t.Fatalf("played should send match line, not congrats: %v", f.bot.sent)
	}
	if f.bot.sent[1] != "Jaedong 2:0 Flash" {
		t.Fatalf("match finished=%q", f.bot.sent[1])
	}

	f.savePlayed(t, true)
	f.notify.Tick(f.ctx, now)
	if len(f.bot.sent) != 3 {
		t.Fatalf("want day_done, sent=%v", f.bot.sent)
	}
	done := f.bot.sent[2]
	if done != "Поздравляем @raynor: Jaedong победил." {
		t.Fatalf("day done=%q", done)
	}

	f.notify.Tick(f.ctx, now)
	if len(f.bot.sent) != 3 {
		t.Fatalf("dedup day_done=%v", f.bot.sent)
	}
}

func TestTelegramNotify_dayDoneNoRosterNoCongrats(t *testing.T) {
	first := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	f := setupNotify(t, first, true)
	f.page.Results[0].Played = true
	f.page.Results[0].ScoreA = intPtr(2)
	f.page.Results[0].ScoreB = intPtr(1)
	if _, _, _, err := f.tours.Save(f.ctx, f.page); err != nil {
		t.Fatal(err)
	}

	f.notify.Tick(f.ctx, first.Add(time.Hour))
	var done string
	for _, m := range f.bot.sent {
		if strings.Contains(m, "победил") {
			done = m
		}
	}
	if done != "Jaedong победил." {
		t.Fatalf("got %q sent=%v", done, f.bot.sent)
	}
	var line string
	for _, m := range f.bot.sent {
		if m == "Jaedong 2:1 Flash" {
			line = m
		}
	}
	if line == "" {
		t.Fatalf("missing match line sent=%v", f.bot.sent)
	}
}

func TestTelegramNotify_sendFailureRetries(t *testing.T) {
	first := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	f := setupNotify(t, first, false)
	f.bot.failNext = 1
	now := first.Add(-time.Minute)
	f.notify.Tick(f.ctx, now)
	if len(f.bot.sent) != 0 {
		t.Fatalf("failed send should not keep message: %v", f.bot.sent)
	}
	f.notify.Tick(f.ctx, now)
	if len(f.bot.sent) != 1 {
		t.Fatalf("retry sent=%v", f.bot.sent)
	}
}

func TestTelegramNotify_idNotInGroupUsesAlias(t *testing.T) {
	first := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	f := setupNotify(t, first, false)
	uid := f.addUser(t, "Nova", int64Ptr(77), str("nova"))
	f.roster(t, uid, f.flashID)

	f.notify.Tick(f.ctx, first)
	if len(f.bot.sent) != 1 {
		t.Fatalf("sent=%v", f.bot.sent)
	}
	if !strings.Contains(f.bot.sent[0], "Удачи командам Nova!") {
		t.Fatalf("want alias, got %s", f.bot.sent[0])
	}
}

func TestTelegramNotify_notificationsDisabledUsesAlias(t *testing.T) {
	first := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	f := setupNotify(t, first, false)
	uid := f.addUser(t, "@Jim", int64Ptr(9001), str("raynor"))
	f.bot.members[9001] = telegram.ChatMember{Status: "member", User: telegram.User{ID: 9001, Username: "raynor"}}
	if err := f.users.UpdateNotificationsEnabled(f.ctx, f.users.DB(), uid, false); err != nil {
		t.Fatal(err)
	}
	f.roster(t, uid, f.jdID)

	f.notify.Tick(f.ctx, first)
	if f.bot.getChatMemberCalls != 0 {
		t.Fatalf("disabled notify must not getChatMember, calls=%d", f.bot.getChatMemberCalls)
	}
	if len(f.bot.sent) != 1 {
		t.Fatalf("sent=%v", f.bot.sent)
	}
	if strings.Contains(f.bot.sent[0], "@raynor") || strings.Contains(f.bot.sent[0], "@Jim") {
		t.Fatalf("want alias without mention, got %s", f.bot.sent[0])
	}
	if !strings.Contains(f.bot.sent[0], "Удачи командам Jim!") {
		t.Fatalf("want stripped alias, got %s", f.bot.sent[0])
	}
}

func TestTelegramNotify_getChatMemberOnlyWhenSending(t *testing.T) {
	first := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	f := setupNotify(t, first, false)
	uid := f.addUser(t, "Jim", int64Ptr(9001), str("raynor"))
	f.bot.members[9001] = telegram.ChatMember{Status: "member", User: telegram.User{ID: 9001, Username: "raynor"}}
	f.roster(t, uid, f.jdID)

	f.notify.Tick(f.ctx, first.Add(-16*time.Minute))
	if f.bot.getChatMemberCalls != 0 {
		t.Fatalf("too early must not getChatMember, calls=%d", f.bot.getChatMemberCalls)
	}

	f.notify.Tick(f.ctx, first)
	if f.bot.getChatMemberCalls != 1 {
		t.Fatalf("prematch send calls=%d", f.bot.getChatMemberCalls)
	}

	f.notify.Tick(f.ctx, first.Add(time.Minute))
	if f.bot.getChatMemberCalls != 1 {
		t.Fatalf("already sent must not getChatMember, calls=%d", f.bot.getChatMemberCalls)
	}
}

func TestTelegramNotify_matchScoreSendsOnChangeNotDuplicate(t *testing.T) {
	first := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	f := setupNotify(t, first, false)
	now := first.Add(time.Hour)

	f.notify.Tick(f.ctx, now)
	if len(f.bot.sent) != 1 || !strings.Contains(f.bot.sent[0], "Сегодня") {
		t.Fatalf("prematch first: %v", f.bot.sent)
	}

	f.savePlayed(t, false)
	f.notify.Tick(f.ctx, now)
	if len(f.bot.sent) != 2 || f.bot.sent[1] != "Jaedong 2:0 Flash" {
		t.Fatalf("first score: %v", f.bot.sent)
	}

	f.notify.Tick(f.ctx, now)
	if len(f.bot.sent) != 2 {
		t.Fatalf("same score must not resend: %v", f.bot.sent)
	}

	f.page.Results[0].ScoreA = intPtr(2)
	f.page.Results[0].ScoreB = intPtr(1)
	if _, _, _, err := f.tours.Save(f.ctx, f.page); err != nil {
		t.Fatal(err)
	}
	f.notify.Tick(f.ctx, now)
	if len(f.bot.sent) != 3 || f.bot.sent[2] != "Jaedong 2:1 Flash" {
		t.Fatalf("new score should send: %v", f.bot.sent)
	}
}

func TestTelegramNotify_skipsZeroZeroScore(t *testing.T) {
	first := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	f := setupNotify(t, first, false)
	now := first.Add(time.Hour)

	f.page.Results[0].Played = true
	f.page.Results[0].ScoreA = intPtr(0)
	f.page.Results[0].ScoreB = intPtr(0)
	if _, _, _, err := f.tours.Save(f.ctx, f.page); err != nil {
		t.Fatal(err)
	}
	f.notify.Tick(f.ctx, now)
	for _, m := range f.bot.sent {
		if strings.Contains(m, "0:0") {
			t.Fatalf("must not send 0:0: %v", f.bot.sent)
		}
	}
}

func int64Ptr(v int64) *int64 { return &v }
