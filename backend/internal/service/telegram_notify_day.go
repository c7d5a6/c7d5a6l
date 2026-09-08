package service

import (
	"strings"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

type namedPlayer struct {
	name string
	link string
}

func utcCalendarDay(iso string) string {
	iso = strings.TrimSpace(iso)
	if iso == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, iso); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	if t, err := time.Parse(time.RFC3339Nano, iso); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	if len(iso) >= 10 && iso[4] == '-' && iso[7] == '-' {
		return iso[:10]
	}
	return ""
}

func parseResultTime(iso string) (time.Time, bool) {
	iso = strings.TrimSpace(iso)
	if iso == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, iso); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse(time.RFC3339Nano, iso); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

func matchesOnDay(results []model.Result, day string) []model.Result {
	out := make([]model.Result, 0)
	for _, r := range results {
		if r.DateTime == nil {
			continue
		}
		if utcCalendarDay(*r.DateTime) == day {
			out = append(out, r)
		}
	}
	return out
}

func firstMatchStart(matches []model.Result) (time.Time, bool) {
	var first time.Time
	ok := false
	for _, m := range matches {
		if m.DateTime == nil {
			continue
		}
		t, parsed := parseResultTime(*m.DateTime)
		if !parsed {
			continue
		}
		if !ok || t.Before(first) {
			first = t
			ok = true
		}
	}
	return first, ok
}

func allMatchesPlayed(matches []model.Result) bool {
	if len(matches) == 0 {
		return false
	}
	for _, m := range matches {
		if !m.Played {
			return false
		}
	}
	return true
}

func normPlayerLink(link *string) string {
	if link == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(*link))
}

func displayPlayerName(name, link *string) string {
	if name != nil {
		if s := strings.TrimSpace(*name); s != "" {
			return s
		}
	}
	if link != nil {
		s := strings.TrimSpace(*link)
		if i := strings.LastIndex(s, "/"); i >= 0 && i+1 < len(s) {
			s = s[i+1:]
		}
		s = strings.ReplaceAll(s, "_", " ")
		return strings.TrimSpace(s)
	}
	return ""
}

func participantLinks(matches []model.Result) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range matches {
		if a := normPlayerLink(participantLink(m.ParticipantA)); a != "" {
			out[a] = struct{}{}
		}
		if b := normPlayerLink(participantLink(m.ParticipantB)); b != "" {
			out[b] = struct{}{}
		}
	}
	return out
}

func participantLink(p *model.Participant) *string {
	if p == nil {
		return nil
	}
	return p.Link
}

func dayPlayerNames(matches []model.Result) []string {
	seen := map[string]struct{}{}
	var names []string
	add := func(p *model.Participant) {
		if p == nil {
			return
		}
		name := displayPlayerName(p.Name, p.Link)
		if name == "" {
			return
		}
		key := normPlayerLink(p.Link)
		if key == "" {
			key = strings.ToLower(name)
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		names = append(names, name)
	}
	for _, m := range matches {
		add(m.ParticipantA)
		add(m.ParticipantB)
	}
	return names
}

func dayPhases(matches []model.Result) []string {
	seen := map[string]struct{}{}
	var phases []string
	for _, m := range matches {
		p := strings.TrimSpace(m.Phase)
		if p == "" {
			continue
		}
		k := strings.ToLower(p)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		phases = append(phases, p)
	}
	return phases
}

func groupWinnersForDay(groups []model.FantasyGroup, matches []model.Result) []namedPlayer {
	groupIDs := map[int64]struct{}{}
	for _, m := range matches {
		if m.GroupID != nil {
			groupIDs[*m.GroupID] = struct{}{}
		}
	}
	var dayLinks map[string]struct{}
	if len(groupIDs) == 0 {
		dayLinks = participantLinks(matches)
	}
	seen := map[string]struct{}{}
	var out []namedPlayer
	for _, g := range groups {
		if len(groupIDs) > 0 {
			if _, ok := groupIDs[g.ID]; !ok {
				continue
			}
		}
		for _, p := range g.Players {
			if !p.IsGroupWinner {
				continue
			}
			link := normPlayerLink(p.Link)
			if link == "" {
				continue
			}
			if dayLinks != nil {
				if _, ok := dayLinks[link]; !ok {
					continue
				}
			}
			if _, ok := seen[link]; ok {
				continue
			}
			name := displayPlayerName(p.Name, p.Link)
			if name == "" {
				continue
			}
			seen[link] = struct{}{}
			out = append(out, namedPlayer{name: name, link: link})
		}
	}
	return out
}

func winnerNames(winners []namedPlayer) []string {
	names := make([]string, 0, len(winners))
	for _, w := range winners {
		names = append(names, w.name)
	}
	return names
}

func winnerLinks(winners []namedPlayer) map[string]struct{} {
	out := make(map[string]struct{}, len(winners))
	for _, w := range winners {
		if w.link != "" {
			out[w.link] = struct{}{}
		}
	}
	return out
}

func operatorTeamsForLinks(teams []model.FantasyTeamRow, links map[string]struct{}) []model.FantasyTeamRow {
	if len(links) == 0 {
		return nil
	}
	out := make([]model.FantasyTeamRow, 0)
	for _, team := range teams {
		for _, m := range team.Members {
			link := normPlayerLink(m.Link)
			if link == "" {
				continue
			}
			if _, ok := links[link]; ok {
				out = append(out, team)
				break
			}
		}
	}
	return out
}
