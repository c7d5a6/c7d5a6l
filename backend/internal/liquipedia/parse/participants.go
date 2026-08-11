package parse

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// Participants extracts tournament entrants: name, profile link, race, excluded.
// Supports modern .participantTable grids, legacy .participantstable, and KCM race-team cards.
func Participants(doc *goquery.Document) ([]model.Participant, error) {
	if list := participantsFromFactionTable(doc); len(list) > 0 {
		return list, nil
	}
	if list := participantsFromLegacyTable(doc); len(list) > 0 {
		return list, nil
	}
	if list := participantsFromRaceTeams(doc); len(list) > 0 {
		return list, nil
	}
	return []model.Participant{}, nil
}

func participantsFromFactionTable(doc *goquery.Document) []model.Participant {
	table := doc.Find(".participantTable").First()
	if table.Length() == 0 {
		return nil
	}

	races := factionHeaderRaces(table)
	if len(races) == 0 {
		return nil
	}

	var out []model.Participant
	seen := map[string]struct{}{}

	table.Find(".participantTable-row").Each(func(_ int, row *goquery.Selection) {
		if row.Find(".participantTable-title").Length() > 0 {
			return
		}
		if row.Find(".participantTable-faction-header").Length() > 0 {
			return
		}

		row.ChildrenFiltered(".participantTable-entry").Each(func(col int, entry *goquery.Selection) {
			if entry.Find(".participantTable-faction-header").Length() > 0 {
				return
			}
			p, ok := participantFromBlock(entry, raceAt(races, col))
			if !ok {
				return
			}
			key := participantKey(p)
			if _, exists := seen[key]; exists {
				return
			}
			seen[key] = struct{}{}
			out = append(out, p)
		})
	})

	return out
}

func factionHeaderRaces(table *goquery.Selection) []string {
	var races []string
	table.Find(".participantTable-faction-header").Each(func(_ int, header *goquery.Selection) {
		race := raceFromClasses(header)
		if race == "" {
			race = normalizeRace(header.Find("img").First().AttrOr("alt", ""))
		}
		if race == "" {
			race = normalizeRace(cleanText(header.Text()))
		}
		if race != "" {
			races = append(races, race)
		}
	})
	return races
}

func participantsFromLegacyTable(doc *goquery.Document) []model.Participant {
	table := doc.Find("table.wikitable.participantstable").First()
	if table.Length() == 0 {
		return nil
	}

	var races []string
	var out []model.Participant
	seen := map[string]struct{}{}

	table.Find("tr").Each(func(_ int, row *goquery.Selection) {
		if row.Find("th").Length() > 0 && row.Find("td").Length() == 0 {
			return
		}

		headers := row.ChildrenFiltered("td.Protoss, td.Terran, td.Zerg, td.Random")
		if headers.Length() > 0 {
			races = nil
			headers.Each(func(_ int, cell *goquery.Selection) {
				race := raceFromClasses(cell)
				if race == "" {
					race = normalizeRace(cell.Find("img").First().AttrOr("alt", ""))
				}
				if race != "" {
					races = append(races, race)
				}
			})
			return
		}

		if len(races) == 0 {
			return
		}

		row.ChildrenFiltered("td").Each(func(col int, cell *goquery.Selection) {
			text := cleanText(cell.Text())
			if text == "" || text == "—" || text == "-" || text == "–" {
				return
			}
			p, ok := participantFromLegacyCell(cell, raceAt(races, col))
			if !ok {
				return
			}
			key := participantKey(p)
			if _, exists := seen[key]; exists {
				return
			}
			seen[key] = struct{}{}
			out = append(out, p)
		})
	})

	return out
}

func participantFromLegacyCell(cell *goquery.Selection, race string) (model.Participant, bool) {
	player := cell.Find(".inline-player").First()
	if player.Length() == 0 {
		player = cell
	}

	link := player.Find("a[href]").First()
	if link.Length() == 0 {
		return model.Participant{}, false
	}

	name := cleanText(link.Text())
	if name == "" || strings.EqualFold(name, "TBD") {
		return model.Participant{}, false
	}

	href, _ := link.Attr("href")
	profile := profileURL(href)
	if profile != nil && isRaceProfileLink(*profile) {
		return model.Participant{}, false
	}

	p := model.Participant{
		Name:     &name,
		Link:     profile,
		Excluded: player.Find("s").Length() > 0 || cell.Find("s").Length() > 0,
	}
	if race != "" {
		r := race
		p.Race = &r
	}
	return p, true
}

func participantsFromRaceTeams(doc *goquery.Document) []model.Participant {
	var out []model.Participant
	seen := map[string]struct{}{}

	doc.Find(".team-participant-card").Each(func(_ int, card *goquery.Selection) {
		race := cleanText(card.Find(".team-participant-card__opponent-compact .name").First().Text())
		if race == "" {
			race = cleanText(card.Find(".team-participant-card__opponent .name").First().Text())
		}
		race = normalizeRace(race)
		if race == "" {
			return
		}

		card.Find(".team-participant-card__member").Each(func(_ int, member *goquery.Selection) {
			player := member.Find(".block-player").First()
			if player.Length() == 0 {
				return
			}

			link := player.Find(".name a[href]").First()
			if link.Length() == 0 {
				link = player.Find("a[href]").First()
			}
			if link.Length() == 0 {
				return
			}

			name := cleanText(link.Text())
			if name == "" || strings.EqualFold(name, "TBD") {
				return
			}

			memberRace := race
			if alt := normalizeRace(player.Find(".race img").First().AttrOr("alt", "")); alt != "" {
				memberRace = alt
			}

			role := cleanText(member.Find(".team-participant-card__member-role-right").First().Text())
			excluded := strings.EqualFold(role, "DNP")

			href, _ := link.Attr("href")
			p := model.Participant{
				Name:     &name,
				Link:     profileURL(href),
				Excluded: excluded,
			}
			if memberRace != "" {
				r := memberRace
				p.Race = &r
			}

			key := participantKey(p)
			if _, exists := seen[key]; exists {
				return
			}
			seen[key] = struct{}{}
			out = append(out, p)
		})
	})

	return out
}

func participantFromBlock(scope *goquery.Selection, race string) (model.Participant, bool) {
	player := scope.Find(".block-player").First()
	if player.Length() == 0 {
		return model.Participant{}, false
	}

	excluded := player.Find("s.name").Length() > 0
	link := player.Find("s.name a[href], span.name a[href], .name a[href]").First()
	if link.Length() == 0 {
		link = player.Find("a[href]").First()
	}
	if link.Length() == 0 {
		return model.Participant{}, false
	}

	name := cleanText(link.Text())
	if name == "" {
		name = cleanText(scope.AttrOr("aria-label", ""))
	}
	if name == "" || strings.EqualFold(name, "TBD") {
		return model.Participant{}, false
	}

	href, _ := link.Attr("href")
	p := model.Participant{
		Name:     &name,
		Link:     profileURL(href),
		Excluded: excluded,
	}
	if race != "" {
		r := race
		p.Race = &r
	} else if alt := normalizeRace(player.Find(".race img").First().AttrOr("alt", "")); alt != "" {
		p.Race = &alt
	}
	return p, true
}

func raceAt(races []string, i int) string {
	if i < 0 || i >= len(races) {
		return ""
	}
	return races[i]
}

func raceFromClasses(sel *goquery.Selection) string {
	class, _ := sel.Attr("class")
	for _, part := range strings.Fields(class) {
		if r := normalizeRace(part); r != "" {
			return r
		}
	}
	return ""
}

func normalizeRace(s string) string {
	s = strings.ToLower(cleanText(s))
	switch {
	case strings.Contains(s, "protoss"):
		return "protoss"
	case strings.Contains(s, "terran"):
		return "terran"
	case strings.Contains(s, "zerg"):
		return "zerg"
	case strings.Contains(s, "random"):
		return "random"
	default:
		return ""
	}
}

func profileURL(href string) *string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") {
		return nil
	}

	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		u, err := url.Parse(href)
		if err != nil || !strings.EqualFold(u.Host, liquipedia.AllowedHost) {
			return nil
		}
		out := u.String()
		return &out
	}

	if !strings.HasPrefix(href, "/") {
		return nil
	}
	out := "https://" + liquipedia.AllowedHost + href
	return &out
}

func isRaceProfileLink(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	base := strings.Trim(u.Path, "/")
	parts := strings.Split(base, "/")
	if len(parts) != 2 {
		return false
	}
	return normalizeRace(parts[1]) != ""
}

func participantKey(p model.Participant) string {
	if p.Link != nil && *p.Link != "" {
		return strings.ToLower(*p.Link)
	}
	if p.Name != nil {
		return strings.ToLower(*p.Name)
	}
	return ""
}
