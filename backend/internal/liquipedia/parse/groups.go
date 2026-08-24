package parse

import (
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// Groups extracts named player pools from standings tables and/or Result.Stage
// paths, then always merges playoff bracket rounds as groups under phase Playoffs.
func Groups(doc *goquery.Document, results []model.Result) ([]model.TournamentGroup, error) {
	groups := groupsFromStandings(doc)
	if len(groups) == 0 {
		groups = groupsFromResults(results)
	}
	groups = mergeGroups(groups, playoffGroupsFromResults(results))
	applyAdvancingFromResults(groups, results)
	if groups == nil {
		groups = []model.TournamentGroup{}
	}
	return groups, nil
}

func groupsFromStandings(doc *goquery.Document) []model.TournamentGroup {
	var (
		out    []model.TournamentGroup
		seen   = map[string]struct{}{}
		h3, h4 string
		order  int
	)

	doc.Find("h3, h4, table, div.group-table").Each(func(_ int, sel *goquery.Selection) {
		switch {
		case sel.Is("h3"):
			h3 = cleanHeading(sel)
			h4 = ""
		case sel.Is("h4"):
			h4 = cleanHeading(sel)
		case sel.Is("table") && isGroupStandingsTable(sel):
			name := groupTableName(sel, h4)
			appendStandingGroup(&out, seen, &order, h3, name, playersFromGroupTable(sel))
		case sel.Is("div") && sel.HasClass("group-table"):
			name := groupDivName(sel, h4)
			appendStandingGroup(&out, seen, &order, h3, name, playersFromGroupDiv(sel))
		}
	})

	return out
}

func appendStandingGroup(
	out *[]model.TournamentGroup,
	seen map[string]struct{},
	order *int,
	phase, name string,
	rows []standingPlayer,
) {
	if name == "" || phase == "" || len(rows) == 0 {
		return
	}
	key := groupKey(phase, name)
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*out = append(*out, model.TournamentGroup{
		Name:      name,
		Phase:     phase,
		SortOrder: *order,
		Players:   applyGroupWinners(rows, true),
	})
	*order++
}

type standingPlayer struct {
	p         model.Participant
	advancing bool
}

func applyGroupWinners(rows []standingPlayer, fromStandings bool) []model.Participant {
	out := make([]model.Participant, 0, len(rows))
	anyMark := false
	for _, r := range rows {
		if r.advancing {
			anyMark = true
			break
		}
	}
	n := len(rows)
	topN := 0
	if fromStandings && !anyMark {
		if n >= 4 {
			topN = 2
		} else if n > 0 {
			topN = 1
		}
	}
	for i, r := range rows {
		p := r.p
		switch {
		case anyMark:
			p.IsWinner = r.advancing
		case i < topN:
			p.IsWinner = true
		}
		out = append(out, p)
	}
	return out
}

func isAdvancingClass(class string) bool {
	for _, part := range strings.Fields(strings.ToLower(class)) {
		switch part {
		case "bg-up", "bg-u", "bg-proceed", "bg-win", "bg-uptodate", "bg-safe":
			return true
		}
	}
	return false
}

func selectionAdvances(sel *goquery.Selection) bool {
	if sel.Length() == 0 {
		return false
	}
	if isAdvancingClass(sel.AttrOr("class", "")) {
		return true
	}
	adv := false
	sel.Find("[class]").EachWithBreak(func(_ int, n *goquery.Selection) bool {
		if isAdvancingClass(n.AttrOr("class", "")) {
			adv = true
			return false
		}
		return true
	})
	return adv
}

func isGroupStandingsTable(sel *goquery.Selection) bool {
	class, _ := sel.Attr("class")
	for _, part := range strings.Fields(strings.ToLower(class)) {
		if part == "grouptable" || part == "group-table" {
			return true
		}
	}
	return false
}

func groupTableName(table *goquery.Selection, h4 string) string {
	if name := cleanText(h4); name != "" {
		return name
	}
	if cap := cleanText(table.Find("caption").First().Text()); cap != "" {
		return cap
	}
	var name string
	table.Find("th").EachWithBreak(func(_ int, th *goquery.Selection) bool {
		text := cleanText(th.Text())
		if text == "" {
			return true
		}
		if strings.HasPrefix(strings.ToLower(text), "group") {
			name = text
			return false
		}
		if name == "" {
			name = text
		}
		return true
	})
	return name
}

func groupDivName(div *goquery.Selection, h4 string) string {
	if title := cleanText(div.Find(".group-table-title").First().Text()); title != "" {
		return title
	}
	return groupTableName(div, h4)
}

func playersFromGroupTable(table *goquery.Selection) []standingPlayer {
	var out []standingPlayer
	seen := map[string]struct{}{}

	table.Find("tr").Each(func(_ int, row *goquery.Selection) {
		appendStandingPlayers(&out, seen, row, row.Find(".block-player"))
	})

	return out
}

func playersFromGroupDiv(div *goquery.Selection) []standingPlayer {
	var out []standingPlayer
	seen := map[string]struct{}{}

	div.Find(".group-table-result-row").Each(func(_ int, row *goquery.Selection) {
		appendStandingPlayers(&out, seen, row, row.Find(".block-player"))
	})
	if len(out) == 0 {
		div.Find(".block-player").Each(func(_ int, block *goquery.Selection) {
			row := block.Closest(".group-table-result-row")
			if row.Length() == 0 {
				row = block.Parent()
			}
			appendStandingPlayers(&out, seen, row, block)
		})
	}

	return out
}

func appendStandingPlayers(out *[]standingPlayer, seen map[string]struct{}, row, blocks *goquery.Selection) {
	rowAdvancing := selectionAdvances(row)
	blocks.Each(func(_ int, block *goquery.Selection) {
		scope := block.Closest("td, th, .group-table-cell, .group-table-entry")
		if scope.Length() == 0 {
			scope = block.Parent()
		}
		p, ok := participantFromBlock(scope, "")
		if !ok {
			return
		}
		key := participantKey(p)
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		advancing := rowAdvancing || selectionAdvances(scope) || selectionAdvances(block)
		*out = append(*out, standingPlayer{p: p, advancing: advancing})
	})
}

func groupsFromResults(results []model.Result) []model.TournamentGroup {
	return collectGroupsFromResults(results, true)
}

func playoffGroupsFromResults(results []model.Result) []model.TournamentGroup {
	return collectGroupsFromResults(results, false)
}

func collectGroupsFromResults(results []model.Result, poolOnly bool) []model.TournamentGroup {
	type bucket struct {
		phase   string
		name    string
		players []model.Participant
		seen    map[string]struct{}
	}

	var order []string
	byKey := map[string]*bucket{}

	add := func(phase, name string, p *model.Participant) {
		if p == nil {
			return
		}
		key := groupKey(phase, name)
		b, ok := byKey[key]
		if !ok {
			b = &bucket{
				phase: phase,
				name:  name,
				seen:  map[string]struct{}{},
			}
			byKey[key] = b
			order = append(order, key)
		}
		pk := participantKey(*p)
		if pk == "" {
			return
		}
		if _, exists := b.seen[pk]; exists {
			return
		}
		b.seen[pk] = struct{}{}
		b.players = append(b.players, *p)
	}

	for _, r := range results {
		if r.Stage == nil {
			continue
		}
		var phase, name string
		var ok bool
		if poolOnly {
			phase, name, ok = parseGroupStage(*r.Stage)
		} else {
			phase, name, ok = parsePlayoffStage(*r.Stage)
		}
		if !ok {
			continue
		}
		add(phase, name, r.ParticipantA)
		add(phase, name, r.ParticipantB)
	}

	out := make([]model.TournamentGroup, 0, len(order))
	for i, key := range order {
		b := byKey[key]
		if len(b.players) == 0 {
			continue
		}
		out = append(out, model.TournamentGroup{
			Name:      b.name,
			Phase:     b.phase,
			SortOrder: i,
			Players:   b.players,
		})
	}
	return out
}

// applyAdvancingFromResults marks dual-tournament advancers (Winner's Match + Final Match)
// on existing pool groups. Standings CSS remains the primary signal; this fills gaps
// when tables omit bg-up / bg-proceed.
func applyAdvancingFromResults(groups []model.TournamentGroup, results []model.Result) {
	if len(groups) == 0 || len(results) == 0 {
		return
	}
	for i := range groups {
		g := &groups[i]
		if strings.Contains(strings.ToLower(g.Phase), "playoff") {
			continue
		}
		adv := dualTournamentAdvancers(results, g.Phase, g.Name)
		if len(adv) == 0 {
			continue
		}
		for j := range g.Players {
			if adv[participantKey(g.Players[j])] {
				g.Players[j].IsWinner = true
			}
		}
	}
}

func dualTournamentAdvancers(results []model.Result, phase, name string) map[string]bool {
	out := map[string]bool{}
	for _, r := range results {
		if !r.Played || r.Stage == nil {
			continue
		}
		p, n, ok := parseGroupStage(*r.Stage)
		if !ok || !strings.EqualFold(p, phase) || !strings.EqualFold(n, name) {
			continue
		}
		if !isDualAdvanceRound(*r.Stage) {
			continue
		}
		w := playedMatchWinner(r)
		if w == nil {
			continue
		}
		if key := participantKey(*w); key != "" {
			out[key] = true
		}
	}
	return out
}

func isDualAdvanceRound(stage string) bool {
	segs := stageSegments(stage)
	if len(segs) == 0 {
		return false
	}
	round := strings.ToLower(cleanText(segs[len(segs)-1]))
	if strings.Contains(round, "loser") || strings.Contains(round, "grand") {
		return false
	}
	if strings.Contains(round, "winner") && strings.Contains(round, "match") {
		return true
	}
	return round == "final match" || round == "finals match"
}

func playedMatchWinner(r model.Result) *model.Participant {
	if r.ScoreA == nil || r.ScoreB == nil {
		return nil
	}
	switch {
	case *r.ScoreA > *r.ScoreB:
		return r.ParticipantA
	case *r.ScoreB > *r.ScoreA:
		return r.ParticipantB
	default:
		return nil
	}
}

func mergeGroups(base, extra []model.TournamentGroup) []model.TournamentGroup {
	seen := map[string]struct{}{}
	out := make([]model.TournamentGroup, 0, len(base)+len(extra))
	for _, g := range base {
		key := groupKey(g.Phase, g.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		g.SortOrder = len(out)
		out = append(out, g)
	}
	for _, g := range extra {
		key := groupKey(g.Phase, g.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		g.SortOrder = len(out)
		out = append(out, g)
	}
	return out
}

// parseGroupStage finds a segment starting with "Group" in a slash-separated stage path.
// Phase is the first segment (e.g. "Round of 24"); name is the Group segment.
func parseGroupStage(stage string) (phase, name string, ok bool) {
	cleaned := stageSegments(stage)
	if len(cleaned) == 0 {
		return "", "", false
	}
	groupIdx := -1
	for i, p := range cleaned {
		lower := strings.ToLower(p)
		if lower == "group" || strings.HasPrefix(lower, "group ") {
			groupIdx = i
			break
		}
	}
	if groupIdx < 0 {
		return "", "", false
	}
	return cleaned[0], cleaned[groupIdx], true
}

// parsePlayoffStage maps bracket stages to phase Playoffs + round name.
func parsePlayoffStage(stage string) (phase, name string, ok bool) {
	if _, _, isGroup := parseGroupStage(stage); isGroup {
		return "", "", false
	}
	cleaned := stageSegments(stage)
	if len(cleaned) == 0 {
		return "", "", false
	}
	if strings.Contains(strings.ToLower(cleaned[0]), "playoff") {
		if len(cleaned) < 2 {
			return "", "", false
		}
		return "Playoffs", cleaned[len(cleaned)-1], true
	}
	name = cleaned[len(cleaned)-1]
	if isBracketRoundName(name) {
		return "Playoffs", name, true
	}
	return "", "", false
}

func isBracketRoundName(s string) bool {
	lower := strings.ToLower(cleanText(s))
	for _, marker := range []string{
		"grand final", "grand finals", "finals", "final",
		"semifinals", "semifinal", "quarterfinals", "quarterfinal",
		"round of 16", "round of 8", "third place", "3rd place",
	} {
		if lower == marker || strings.HasPrefix(lower, marker) {
			return true
		}
	}
	return false
}

func stageSegments(stage string) []string {
	var cleaned []string
	for _, p := range strings.Split(stage, "/") {
		p = cleanText(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return cleaned
}

// StagePhaseRound splits a Liquipedia stage path into phase + round for persistence.
func StagePhaseRound(stage string) (phase, round string) {
	if stage == "" {
		return "", ""
	}
	if p, name, ok := parseGroupStage(stage); ok {
		return p, name
	}
	if p, name, ok := parsePlayoffStage(stage); ok {
		return p, name
	}
	cleaned := stageSegments(stage)
	if len(cleaned) == 0 {
		return "", ""
	}
	if len(cleaned) == 1 {
		return cleaned[0], ""
	}
	return cleaned[0], cleaned[len(cleaned)-1]
}

func groupKey(phase, name string) string {
	return strings.ToLower(cleanText(phase)) + "\x00" + strings.ToLower(cleanText(name))
}
