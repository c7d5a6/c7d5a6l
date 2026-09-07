package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/telegram"
)

const DefaultTelegramPrematchLead = 15 * time.Minute

// GroupBot is the Telegram client surface used for group notices.
type GroupBot interface {
	Configured() bool
	SendGroup(ctx context.Context, text string) error
	GetChatMember(ctx context.Context, userID int64) (*telegram.ChatMember, error)
}

// TelegramNotify sends fantasy match-day messages to the configured group.
type TelegramNotify struct {
	db      *sql.DB
	fantasy *Fantasy
	users   *repository.User
	notices *repository.TelegramNotice
	bot     GroupBot
	lead    time.Duration
}

func NewTelegramNotify(
	db *sql.DB,
	fantasy *Fantasy,
	users *repository.User,
	notices *repository.TelegramNotice,
	bot GroupBot,
	lead time.Duration,
) *TelegramNotify {
	if lead <= 0 {
		lead = DefaultTelegramPrematchLead
	}
	if notices == nil {
		notices = repository.NewTelegramNotice()
	}
	return &TelegramNotify{
		db:      db,
		fantasy: fantasy,
		users:   users,
		notices: notices,
		bot:     bot,
		lead:    lead,
	}
}

// Tick checks started leagues for prematch and day-done sends.
func (s *TelegramNotify) Tick(ctx context.Context, now time.Time) {
	if s == nil || s.bot == nil || !s.bot.Configured() || s.fantasy == nil {
		return
	}
	now = now.UTC()
	leagues, err := s.fantasy.ListLeagues(ctx)
	if err != nil {
		log.Printf("telegram notify: list leagues: %v", err)
		return
	}
	for _, league := range leagues {
		if !league.Started || league.Finished {
			continue
		}
		s.maybePrematch(ctx, league, now)
		s.maybeMatchFinished(ctx, league, now)
		s.maybeDayDone(ctx, league, now)
	}
}

// OnTournamentSaved retries day-done after results are upserted.
func (s *TelegramNotify) OnTournamentSaved(ctx context.Context, tournamentID int64) {
	if s == nil || s.bot == nil || !s.bot.Configured() || s.fantasy == nil || tournamentID == 0 {
		return
	}
	league, err := s.fantasy.LeagueForTournament(ctx, tournamentID)
	if err != nil {
		log.Printf("telegram notify: league for tournament %d: %v", tournamentID, err)
		return
	}
	if league == nil || !league.Started || league.Finished {
		return
	}
	s.maybeMatchFinished(ctx, *league, time.Now().UTC())
	s.maybeDayDone(ctx, *league, time.Now().UTC())
}

func (s *TelegramNotify) maybePrematch(ctx context.Context, league model.FantasyLeague, now time.Time) {
	day := now.Format("2006-01-02")
	board, err := s.fantasy.MatchBoard(ctx, league.ID)
	if err != nil {
		log.Printf("telegram notify: match board league=%d: %v", league.ID, err)
		return
	}
	matches := matchesOnDay(board.Results, day)
	first, ok := firstMatchStart(matches)
	if !ok {
		return
	}
	if now.Before(first.Add(-s.lead)) {
		return
	}
	players := dayPlayerNames(matches)
	if len(players) == 0 {
		return
	}
	teams, err := s.fantasy.ListTeams(ctx, league.ID)
	if err != nil {
		log.Printf("telegram notify: teams league=%d: %v", league.ID, err)
		return
	}
	ops, err := s.formatOperators(ctx, operatorTeamsForLinks(teams, participantLinks(matches)))
	if err != nil {
		log.Printf("telegram notify: operators league=%d: %v", league.ID, err)
		return
	}
	text := formatPrematch(strings.Join(dayPhases(matches), ", "), players, ops)
	s.sendOnce(ctx, repository.TelegramNoticePrematch, league.ID, day, 0, "", now, text)
}

func (s *TelegramNotify) maybeDayDone(ctx context.Context, league model.FantasyLeague, now time.Time) {
	day := now.Format("2006-01-02")
	board, err := s.fantasy.MatchBoard(ctx, league.ID)
	if err != nil {
		log.Printf("telegram notify: match board league=%d: %v", league.ID, err)
		return
	}
	matches := matchesOnDay(board.Results, day)
	if !allMatchesPlayed(matches) {
		return
	}
	winners := groupWinnersForDay(board.Groups, matches)
	if len(winners) == 0 {
		return
	}
	teams, err := s.fantasy.ListTeams(ctx, league.ID)
	if err != nil {
		log.Printf("telegram notify: teams league=%d: %v", league.ID, err)
		return
	}
	ops, err := s.formatOperators(ctx, operatorTeamsForLinks(teams, winnerLinks(winners)))
	if err != nil {
		log.Printf("telegram notify: operators league=%d: %v", league.ID, err)
		return
	}
	text := formatDayDone(ops, winnerNames(winners))
	s.sendOnce(ctx, repository.TelegramNoticeDayDone, league.ID, day, 0, "", now, text)
}

func (s *TelegramNotify) maybeMatchFinished(ctx context.Context, league model.FantasyLeague, now time.Time) {
	day := now.Format("2006-01-02")
	board, err := s.fantasy.MatchBoard(ctx, league.ID)
	if err != nil {
		log.Printf("telegram notify: match board league=%d: %v", league.ID, err)
		return
	}
	for _, m := range matchesOnDay(board.Results, day) {
		text, scoreText, ok := formatMatchScore(m)
		if !ok {
			continue
		}
		s.sendOnce(ctx, repository.TelegramNoticeMatchFinished, league.ID, day, m.ID, scoreText, now, text)
	}
}

func (s *TelegramNotify) sendOnce(ctx context.Context, kind string, leagueID int64, day string, resultID int64, scoreText string, now time.Time, text string) {
	claimed, err := s.notices.Claim(ctx, s.db, kind, leagueID, day, resultID, scoreText, now.Format(time.RFC3339Nano))
	if err != nil {
		log.Printf("telegram notify: claim %s league=%d: %v", kind, leagueID, err)
		return
	}
	if !claimed {
		return
	}
	debuglog.Printf("telegram notify send kind=%s league=%d day=%s result=%d score=%s", kind, leagueID, day, resultID, scoreText)
	if err := s.bot.SendGroup(ctx, text); err != nil {
		log.Printf("telegram notify: send %s league=%d: %v", kind, leagueID, err)
		if relErr := s.notices.Release(ctx, s.db, kind, leagueID, day, resultID, scoreText); relErr != nil {
			log.Printf("telegram notify: release %s league=%d: %v", kind, leagueID, relErr)
		}
		return
	}
	log.Printf("telegram notify: sent %s league=%d day=%s result=%d score=%s", kind, leagueID, day, resultID, scoreText)
}

func (s *TelegramNotify) formatOperators(ctx context.Context, teams []model.FantasyTeamRow) ([]string, error) {
	if s.users == nil {
		out := make([]string, 0, len(teams))
		for _, team := range teams {
			out = append(out, OperatorMention(team.UserAlias, nil, nil, false))
		}
		return out, nil
	}
	cache := map[int64]bool{}
	out := make([]string, 0, len(teams))
	seen := map[int64]struct{}{}
	for _, team := range teams {
		if _, ok := seen[team.UserID]; ok {
			continue
		}
		seen[team.UserID] = struct{}{}
		u, err := s.users.GetByID(ctx, s.db, team.UserID)
		if err != nil {
			return nil, fmt.Errorf("user %d: %w", team.UserID, err)
		}
		if u == nil {
			out = append(out, OperatorMention(team.UserAlias, nil, nil, false))
			continue
		}
		inGroup := false
		if u.TelegramID != nil {
			inGroup = s.memberInGroup(ctx, *u.TelegramID, cache)
		}
		out = append(out, OperatorMention(u.Alias, u.TelegramID, u.TelegramUsername, inGroup))
	}
	return out, nil
}

func (s *TelegramNotify) memberInGroup(ctx context.Context, telegramID int64, cache map[int64]bool) bool {
	if v, ok := cache[telegramID]; ok {
		return v
	}
	in := false
	if s.bot != nil {
		m, err := s.bot.GetChatMember(ctx, telegramID)
		in = err == nil && m != nil && m.InChat()
	}
	cache[telegramID] = in
	return in
}

// OperatorMention renders an operator for a group message.
func OperatorMention(alias string, telegramID *int64, username *string, inGroup bool) string {
	alias = strings.TrimSpace(alias)
	uname := ""
	if username != nil {
		uname = strings.TrimPrefix(strings.TrimSpace(*username), "@")
	}
	if telegramID != nil {
		if inGroup && uname != "" {
			return "@" + uname
		}
		if alias != "" {
			return alias
		}
		if uname != "" {
			return "@" + uname
		}
		return ""
	}
	if uname != "" {
		return "@" + uname
	}
	return alias
}

func formatPrematch(phase string, players, operators []string) string {
	list := strings.Join(players, ", ")
	var line1 string
	if strings.TrimSpace(phase) == "" {
		line1 = fmt.Sprintf("Сегодня играют %s.", list)
	} else {
		line1 = fmt.Sprintf("Сегодня в %s играют %s.", phase, list)
	}
	if len(operators) == 0 {
		return line1
	}
	return line1 + "\n" + fmt.Sprintf("Удачи командам %s!", strings.Join(operators, ", "))
}

func formatDayDone(operators, players []string) string {
	list := strings.Join(players, ", ")
	verb := "победили"
	if len(players) == 1 {
		verb = "победил"
	}
	if len(operators) == 0 {
		return fmt.Sprintf("%s %s.", list, verb)
	}
	return fmt.Sprintf("Поздравляем %s: %s %s.", strings.Join(operators, ", "), list, verb)
}

func formatMatchScore(m model.Result) (line, scoreText string, ok bool) {
	if m.ID == 0 || m.ScoreA == nil || m.ScoreB == nil {
		return "", "", false
	}
	a := ""
	b := ""
	if m.ParticipantA != nil {
		a = displayPlayerName(m.ParticipantA.Name, m.ParticipantA.Link)
	}
	if m.ParticipantB != nil {
		b = displayPlayerName(m.ParticipantB.Name, m.ParticipantB.Link)
	}
	if a == "" || b == "" || strings.EqualFold(a, "TBD") || strings.EqualFold(b, "TBD") {
		return "", "", false
	}
	if *m.ScoreA == 0 && *m.ScoreB == 0 {
		return "", "", false
	}
	scoreText = fmt.Sprintf("%d:%d", *m.ScoreA, *m.ScoreB)
	return fmt.Sprintf("%s %s %s", a, scoreText, b), scoreText, true
}
