package parse

import (
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// PlayerCounts extracts total players and Protoss/Terran/Zerg distribution.
// Prefers the infobox "Number of Players" row; falls back to race-team participant cards.
func PlayerCounts(doc *goquery.Document) (*model.PlayerCounts, error) {
	if counts := playerCountsFromInfobox(doc); counts != nil {
		return counts, nil
	}
	return playerCountsFromRaceTeams(doc), nil
}

func playerCountsFromInfobox(doc *goquery.Document) *model.PlayerCounts {
	var label *goquery.Selection
	doc.Find(".fo-nttax-infobox .infobox-description").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		if normalizeLabel(sel.Text()) == "number of players" {
			label = sel
			return false
		}
		return true
	})
	if label == nil || label.Length() == 0 {
		return nil
	}

	counts := &model.PlayerCounts{}
	if total := parseIntPtr(cleanText(label.Next().Text())); total != nil {
		counts.Total = total
	}

	row := label.Parent().Next()
	if row.Length() == 0 {
		if counts.Total == nil {
			return nil
		}
		return counts
	}

	row.Find(".infobox-cell-3").Each(func(_ int, cell *goquery.Selection) {
		race := strings.ToLower(strings.TrimSpace(cell.Find("img").First().AttrOr("alt", "")))
		n := parseIntPtr(cleanText(cell.Text()))
		if n == nil {
			return
		}
		switch race {
		case "protoss":
			counts.Protoss = n
		case "terran":
			counts.Terran = n
		case "zerg":
			counts.Zerg = n
		}
	})

	if counts.Total == nil && counts.Protoss == nil && counts.Terran == nil && counts.Zerg == nil {
		return nil
	}
	return counts
}

func playerCountsFromRaceTeams(doc *goquery.Document) *model.PlayerCounts {
	counts := &model.PlayerCounts{}
	found := false

	doc.Find(".team-participant-card").Each(func(_ int, card *goquery.Selection) {
		raceName := cleanText(card.Find(".team-participant-card__opponent-compact .name").First().Text())
		if raceName == "" {
			raceName = cleanText(card.Find(".team-participant-card__opponent .name").First().Text())
		}
		raceName = strings.ToLower(raceName)
		if raceName != "protoss" && raceName != "terran" && raceName != "zerg" {
			return
		}

		seen := map[string]struct{}{}
		card.Find(".block-player .name").Each(func(_ int, nameSel *goquery.Selection) {
			name := cleanText(nameSel.Text())
			if name == "" || strings.EqualFold(name, "TBD") {
				return
			}
			seen[strings.ToLower(name)] = struct{}{}
		})
		n := len(seen)
		if n == 0 {
			return
		}
		found = true
		switch raceName {
		case "protoss":
			counts.Protoss = &n
		case "terran":
			counts.Terran = &n
		case "zerg":
			counts.Zerg = &n
		}
	})

	if !found {
		return nil
	}

	total := 0
	for _, p := range []*int{counts.Protoss, counts.Terran, counts.Zerg} {
		if p != nil {
			total += *p
		}
	}
	counts.Total = &total
	return counts
}

func parseIntPtr(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}
