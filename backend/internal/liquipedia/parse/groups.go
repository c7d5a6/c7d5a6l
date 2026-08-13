package parse

import (
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// Groups extracts named player pools from standings tables, falling back to
// Result.Stage paths that contain a "Group …" segment when no tables are found.
func Groups(doc *goquery.Document, results []model.Result) ([]model.TournamentGroup, error) {
	groups := groupsFromStandings(doc)
	if len(groups) == 0 {
		groups = groupsFromResults(results)
	}
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

	doc.Find("h3, h4, table").Each(func(_ int, sel *goquery.Selection) {
		switch {
		case sel.Is("h3"):
			h3 = cleanHeading(sel)
			h4 = ""
		case sel.Is("h4"):
			h4 = cleanHeading(sel)
		case sel.Is("table") && isGroupStandingsTable(sel):
			name := groupTableName(sel, h4)
			phase := h3
			if name == "" || phase == "" {
				return
			}
			players := playersFromGroupTable(sel)
			if len(players) == 0 {
				return
			}
			key := groupKey(phase, name)
			if _, ok := seen[key]; ok {
				return
			}
			seen[key] = struct{}{}
			out = append(out, model.TournamentGroup{
				Name:      name,
				Phase:     phase,
				SortOrder: order,
				Players:   players,
			})
			order++
		}
	})

	return out
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

func playersFromGroupTable(table *goquery.Selection) []model.Participant {
	var out []model.Participant
	seen := map[string]struct{}{}

	table.Find("tr").Each(func(_ int, row *goquery.Selection) {
		row.Find(".block-player").Each(func(_ int, block *goquery.Selection) {
			scope := block.Closest("td, th")
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
			out = append(out, p)
		})
	})

	return out
}

func groupsFromResults(results []model.Result) []model.TournamentGroup {
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
		phase, name, ok := parseGroupStage(*r.Stage)
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

// parseGroupStage finds a segment starting with "Group" in a slash-separated stage path.
// Phase is the first segment (e.g. "Round of 24"); name is the Group segment.
func parseGroupStage(stage string) (phase, name string, ok bool) {
	parts := strings.Split(stage, "/")
	var cleaned []string
	for _, p := range parts {
		p = cleanText(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
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

func groupKey(phase, name string) string {
	return strings.ToLower(cleanText(phase)) + "\x00" + strings.ToLower(cleanText(name))
}
