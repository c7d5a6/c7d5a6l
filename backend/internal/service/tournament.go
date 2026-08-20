package service

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
)

// PlayerPageFetcher loads a player page by Liquipedia link (used when importing missing participants).
type PlayerPageFetcher interface {
	FetchPlayerPage(ctx context.Context, link string) (model.PlayerPage, error)
}

// Tournament orchestrates tournament sync and transactional save.
type Tournament struct {
	db      *sql.DB
	tours   *repository.Tournament
	players *repository.Player
	fetcher PlayerPageFetcher
	assets  AssetFetcher
}

func NewTournament(
	db *sql.DB,
	tournaments *repository.Tournament,
	players *repository.Player,
	fetcher PlayerPageFetcher,
	assets AssetFetcher,
) *Tournament {
	return &Tournament{db: db, tours: tournaments, players: players, fetcher: fetcher, assets: assets}
}

// ListSummaries returns lightweight tournament rows for pickers.
func (s *Tournament) ListSummaries(ctx context.Context) ([]model.TournamentSummary, error) {
	return s.tours.ListSummaries(ctx, s.db)
}

// SyncStatus compares parsed tournament to DB (presence of players by link; no player field diffs).
func (s *Tournament) SyncStatus(ctx context.Context, page model.TournamentPage) (model.TournamentSync, error) {
	stored, err := s.tours.GetByLink(ctx, s.db, page.Link)
	if err != nil {
		return model.TournamentSync{}, err
	}

	playerStatuses, err := s.playerStatuses(ctx, page.Participants)
	if err != nil {
		return model.TournamentSync{}, err
	}

	var storedPage *model.TournamentPage
	var storedParticipants []model.Participant
	if stored != nil {
		p := stored.Page
		storedPage = &p
		storedParticipants = stored.Participants
	}
	return CompareTournament(page, storedPage, storedParticipants, playerStatuses), nil
}

func (s *Tournament) playerStatuses(ctx context.Context, participants []model.Participant) ([]model.TournamentPlayerStatus, error) {
	out := make([]model.TournamentPlayerStatus, 0, len(participants))
	for _, p := range participants {
		st := model.TournamentPlayerStatus{
			Name:     p.Name,
			Link:     p.Link,
			Race:     p.Race,
			Excluded: p.Excluded,
		}
		link := strings.TrimSpace(nullStr(p.Link))
		if link == "" {
			reason := "no link"
			st.SkipReason = &reason
			out = append(out, st)
			continue
		}
		exists, err := s.players.ExistsByLink(ctx, s.db, link)
		if err != nil {
			return nil, err
		}
		st.InDatabase = exists
		st.WillImport = !exists
		out = append(out, st)
	}
	return out, nil
}

// Save fetches missing players (outside TX), then upserts players + tournament + roster in one TX.
func (s *Tournament) Save(ctx context.Context, page model.TournamentPage) (model.TournamentPage, model.TournamentSync, error) {
	if page.Link == "" {
		return model.TournamentPage{}, model.TournamentSync{}, fmt.Errorf("tournament link is required")
	}
	debuglog.Printf("service.Tournament.Save link=%s participants=%d", page.Link, len(page.Participants))

	toImport, err := s.collectMissingLinks(ctx, page.Participants)
	if err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, err
	}

	imported := make([]model.PlayerPage, 0, len(toImport))
	portraits := make([]*repository.PortraitBlob, 0, len(toImport))
	for _, link := range toImport {
		if s.fetcher == nil {
			return model.TournamentPage{}, model.TournamentSync{}, fmt.Errorf("player fetcher not configured")
		}
		debuglog.Printf("service.Tournament.Save fetch player link=%s", link)
		pp, err := s.fetcher.FetchPlayerPage(ctx, link)
		if err != nil {
			return model.TournamentPage{}, model.TournamentSync{}, fmt.Errorf("fetch player %s: %w", link, err)
		}
		canonical, err := liquipedia.ValidateURL(link)
		if err != nil {
			return model.TournamentPage{}, model.TournamentSync{}, fmt.Errorf("player link %s: %w", link, err)
		}
		pp.Link = canonical
		portrait, err := resolvePortrait(ctx, s.assets, pp, nil)
		if err != nil {
			return model.TournamentPage{}, model.TournamentSync{}, fmt.Errorf("portrait %s: %w", link, err)
		}
		imported = append(imported, pp)
		portraits = append(portraits, portrait)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for i, pp := range imported {
		if err := s.players.Upsert(ctx, tx, pp, portraits[i]); err != nil {
			return model.TournamentPage{}, model.TournamentSync{}, fmt.Errorf("upsert imported player: %w", err)
		}
	}

	tournamentID, err := s.tours.Upsert(ctx, tx, page)
	if err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, err
	}

	entries, err := s.buildRosterEntries(ctx, tx, page.Participants)
	if err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, err
	}
	if err := s.tours.ReplaceRoster(ctx, tx, tournamentID, entries); err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, err
	}

	groupEntries := buildGroupEntries(page.Groups)
	if err := s.tours.ReplaceGroups(ctx, tx, tournamentID, groupEntries); err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, err
	}

	resultEntries, err := s.buildResultEntries(ctx, tx, tournamentID, page.Results)
	if err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, err
	}
	if err := s.tours.UpsertResults(ctx, tx, tournamentID, resultEntries); err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, fmt.Errorf("commit: %w", err)
	}

	stored, err := s.tours.GetByLink(ctx, s.db, page.Link)
	if err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, err
	}
	if stored == nil {
		return model.TournamentPage{}, model.TournamentSync{}, fmt.Errorf("tournament missing after save")
	}

	out := stored.Page
	if out.Results == nil {
		out.Results = []model.Result{}
	}
	if out.Groups == nil {
		out.Groups = []model.TournamentGroup{}
	}

	playerStatuses, err := s.playerStatuses(ctx, out.Participants)
	if err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, err
	}
	sync := CompareTournament(out, &out, out.Participants, playerStatuses)
	return out, sync, nil
}

// ListDueRefresh returns tournaments with past-due unplayed matches today (UTC).
func (s *Tournament) ListDueRefresh(ctx context.Context, nowUTC time.Time) ([]repository.TournamentDueRefresh, error) {
	return s.tours.ListTournamentsDueRefresh(ctx, s.db, nowUTC)
}

// RefreshFromHTML parses tournament HTML and saves (roster/groups/results upsert).
func (s *Tournament) RefreshFromHTML(ctx context.Context, link, html string) (model.TournamentPage, error) {
	page, err := parse.Tournament(link, html)
	if err != nil {
		return model.TournamentPage{}, err
	}
	saved, _, err := s.Save(ctx, page)
	return saved, err
}

func (s *Tournament) collectMissingLinks(ctx context.Context, participants []model.Participant) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string
	for _, p := range participants {
		link := strings.TrimSpace(nullStr(p.Link))
		if link == "" {
			continue
		}
		key := strings.ToLower(link)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		exists, err := s.players.ExistsByLink(ctx, s.db, link)
		if err != nil {
			return nil, err
		}
		if !exists {
			out = append(out, link)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *Tournament) buildRosterEntries(ctx context.Context, q repository.DBTX, participants []model.Participant) ([]repository.RosterEntry, error) {
	entries := make([]repository.RosterEntry, 0, len(participants))
	for _, p := range participants {
		link := strings.TrimSpace(nullStr(p.Link))
		if link == "" {
			continue
		}
		playerID, err := s.players.IDByLink(ctx, q, link)
		if err != nil {
			return nil, err
		}
		if playerID == 0 {
			return nil, fmt.Errorf("player not in database for link %s", link)
		}
		race := strings.TrimSpace(nullStr(p.Race))
		if race == "" {
			return nil, fmt.Errorf("participant race required for link %s", link)
		}
		aliasName := strings.TrimSpace(nullStr(p.Name))
		if aliasName == "" {
			aliasName = link
		}
		raceID, err := s.players.EnsureRaceID(ctx, q, playerID, race)
		if err != nil {
			return nil, err
		}
		aliasID, err := s.players.EnsureAliasID(ctx, q, playerID, aliasName)
		if err != nil {
			return nil, err
		}
		entries = append(entries, repository.RosterEntry{
			PlayerRaceID:  raceID,
			PlayerAliasID: aliasID,
			Excluded:      p.Excluded,
		})
	}
	return entries, nil
}

// CompareTournament builds sync metadata for a parsed page vs optional stored row.
func CompareTournament(
	parsed model.TournamentPage,
	stored *model.TournamentPage,
	storedParticipants []model.Participant,
	playerStatuses []model.TournamentPlayerStatus,
) model.TournamentSync {
	if stored == nil {
		return model.TournamentSync{
			Exists:  false,
			Same:    false,
			Action:  model.TournamentActionAdd,
			Players: playerStatuses,
		}
	}

	var changes []model.FieldChange
	if !strPtrEqual(parsed.Name, stored.Name) {
		changes = append(changes, model.FieldChange{Field: "name", Before: strOrNil(stored.Name), After: strOrNil(parsed.Name)})
	}
	if !strPtrEqual(parsed.StartDate, stored.StartDate) {
		changes = append(changes, model.FieldChange{Field: "startDate", Before: strOrNil(stored.StartDate), After: strOrNil(parsed.StartDate)})
	}
	if !strPtrEqual(parsed.EndDate, stored.EndDate) {
		changes = append(changes, model.FieldChange{Field: "endDate", Before: strOrNil(stored.EndDate), After: strOrNil(parsed.EndDate)})
	}
	if !strPtrEqual(parsed.LiquipediaTier, stored.LiquipediaTier) {
		changes = append(changes, model.FieldChange{Field: "tier", Before: strOrNil(stored.LiquipediaTier), After: strOrNil(parsed.LiquipediaTier)})
	}
	if !intPtrEqual(playerCountTotal(parsed.PlayerCounts), playerCountTotal(stored.PlayerCounts)) {
		changes = append(changes, model.FieldChange{
			Field:  "playerCount",
			Before: intOrNil(playerCountTotal(stored.PlayerCounts)),
			After:  intOrNil(playerCountTotal(parsed.PlayerCounts)),
		})
	}
	if !boolPtrEqual(parsed.Finished, stored.Finished) {
		changes = append(changes, model.FieldChange{
			Field:  "finished",
			Before: boolOrNil(stored.Finished),
			After:  boolOrNil(parsed.Finished),
		})
	}
	if !rosterEqual(parsed.Participants, storedParticipants) {
		changes = append(changes, model.FieldChange{
			Field:  "players",
			Before: rosterKeys(storedParticipants),
			After:  rosterKeys(parsed.Participants),
		})
	}
	if !groupsEqual(parsed.Groups, stored.Groups) {
		changes = append(changes, model.FieldChange{
			Field:  "groups",
			Before: groupSummaries(stored.Groups),
			After:  groupSummaries(parsed.Groups),
		})
	}
	if !resultsEqual(parsed.Results, stored.Results) {
		changes = append(changes, model.FieldChange{
			Field:  "results",
			Before: resultsSummaries(stored.Results),
			After:  resultsSummaries(parsed.Results),
		})
	}

	willImport := false
	for _, st := range playerStatuses {
		if st.WillImport {
			willImport = true
			break
		}
	}

	same := len(changes) == 0 && !willImport
	action := model.TournamentActionNone
	if !same {
		action = model.TournamentActionUpdate
	}
	storedCopy := *stored
	return model.TournamentSync{
		Exists:  true,
		Same:    same,
		Action:  action,
		Stored:  &storedCopy,
		Changes: changes,
		Players: playerStatuses,
	}
}

func playerCountTotal(c *model.PlayerCounts) *int {
	if c == nil {
		return nil
	}
	return c.Total
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func intOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func boolOrNil(p *bool) any {
	if p == nil {
		return nil
	}
	return *p
}

func rosterKey(p model.Participant) string {
	link := strings.ToLower(strings.TrimSpace(nullStr(p.Link)))
	name := strings.ToLower(strings.TrimSpace(nullStr(p.Name)))
	race := strings.ToLower(strings.TrimSpace(nullStr(p.Race)))
	id := link
	if id == "" {
		id = name
	}
	ex := "0"
	if p.Excluded {
		ex = "1"
	}
	return id + "|" + race + "|" + ex
}

func rosterKeys(list []model.Participant) []string {
	out := make([]string, 0, len(list))
	for _, p := range list {
		out = append(out, rosterKey(p))
	}
	sort.Strings(out)
	return out
}

func rosterEqual(a, b []model.Participant) bool {
	ka := rosterKeys(a)
	kb := rosterKeys(b)
	if len(ka) != len(kb) {
		return false
	}
	for i := range ka {
		if ka[i] != kb[i] {
			return false
		}
	}
	return true
}

func buildGroupEntries(groups []model.TournamentGroup) []repository.GroupEntry {
	out := make([]repository.GroupEntry, 0, len(groups))
	for i, g := range groups {
		sortOrder := g.SortOrder
		if sortOrder == 0 {
			sortOrder = i
		}
		entry := repository.GroupEntry{
			Name:        g.Name,
			Phase:       g.Phase,
			SortOrder:   sortOrder,
			PlayerLinks: make([]string, 0, len(g.Players)),
		}
		for _, p := range g.Players {
			link := strings.TrimSpace(nullStr(p.Link))
			if link != "" {
				entry.PlayerLinks = append(entry.PlayerLinks, link)
			}
		}
		out = append(out, entry)
	}
	return out
}

func groupSummaries(groups []model.TournamentGroup) []string {
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		out = append(out, strings.ToLower(strings.TrimSpace(g.Phase))+"|"+strings.ToLower(strings.TrimSpace(g.Name)))
	}
	sort.Strings(out)
	return out
}

func groupsEqual(a, b []model.TournamentGroup) bool {
	ka := groupSummaries(a)
	kb := groupSummaries(b)
	if len(ka) != len(kb) {
		return false
	}
	for i := range ka {
		if ka[i] != kb[i] {
			return false
		}
	}
	return true
}

func (s *Tournament) buildResultEntries(ctx context.Context, q repository.DBTX, tournamentID int64, results []model.Result) ([]repository.ResultEntry, error) {
	groupIDs, err := s.tours.GroupIDByPhaseName(ctx, q, tournamentID)
	if err != nil {
		return nil, err
	}
	out := make([]repository.ResultEntry, 0, len(results))
	pairSeen := make(map[[2]int64]int)
	for _, r := range results {
		if r.ParticipantA == nil || r.ParticipantB == nil {
			continue
		}
		linkA := strings.TrimSpace(nullStr(r.ParticipantA.Link))
		linkB := strings.TrimSpace(nullStr(r.ParticipantB.Link))
		if linkA == "" || linkB == "" {
			continue
		}
		idA, err := s.tours.TournamentPlayerIDByLink(ctx, q, tournamentID, linkA)
		if err != nil {
			return nil, err
		}
		idB, err := s.tours.TournamentPlayerIDByLink(ctx, q, tournamentID, linkB)
		if err != nil {
			return nil, err
		}
		if idA == 0 || idB == 0 {
			debuglog.Printf("UpsertResults skip unresolved players a=%s b=%s", linkA, linkB)
			continue
		}
		phase, round := r.Phase, r.Round
		if phase == "" && r.Stage != nil {
			phase, round = parse.StagePhaseRound(*r.Stage)
		}
		var groupID *int64
		key := strings.ToLower(strings.TrimSpace(phase)) + "\x00" + strings.ToLower(strings.TrimSpace(round))
		if id, ok := groupIDs[key]; ok {
			groupID = &id
		}
		lo, hi := idA, idB
		if lo > hi {
			lo, hi = hi, lo
		}
		pairKey := [2]int64{lo, hi}
		pairIndex := pairSeen[pairKey]
		pairSeen[pairKey] = pairIndex + 1
		out = append(out, repository.ResultEntry{
			Phase:               phase,
			Round:               round,
			TournamentGroupID:   groupID,
			TournamentPlayerAID: idA,
			TournamentPlayerBID: idB,
			PairIndex:           pairIndex,
			ScoreA:              r.ScoreA,
			ScoreB:              r.ScoreB,
			Played:              r.Played,
			PlayedAt:            r.DateTime,
			SortOrder:           r.Order,
		})
	}
	return out, nil
}

func resultsSummaries(results []model.Result) []string {
	out := make([]string, 0, len(results))
	for _, r := range results {
		a := strings.ToLower(strings.TrimSpace(nullStr(ptrLink(r.ParticipantA))))
		b := strings.ToLower(strings.TrimSpace(nullStr(ptrLink(r.ParticipantB))))
		if a > b {
			a, b = b, a
		}
		sa, sb := "-", "-"
		if r.ScoreA != nil {
			sa = fmt.Sprintf("%d", *r.ScoreA)
		}
		if r.ScoreB != nil {
			sb = fmt.Sprintf("%d", *r.ScoreB)
		}
		played := "0"
		if r.Played {
			played = "1"
		}
		out = append(out, a+"|"+b+"|"+played+"|"+sa+":"+sb)
	}
	sort.Strings(out)
	return out
}

func ptrLink(p *model.Participant) *string {
	if p == nil {
		return nil
	}
	return p.Link
}

func resultsEqual(a, b []model.Result) bool {
	ka := resultsSummaries(a)
	kb := resultsSummaries(b)
	if len(ka) != len(kb) {
		return false
	}
	for i := range ka {
		if ka[i] != kb[i] {
			return false
		}
	}
	return true
}
