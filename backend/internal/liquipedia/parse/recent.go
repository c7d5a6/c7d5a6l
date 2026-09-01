package parse

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

var isoDateRE = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

// RecentTournaments extracts tournament rows from the Leagues/Recent_Tournaments listing.
func RecentTournaments(html string) ([]model.TournamentListing, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var out []model.TournamentListing
	seen := map[string]struct{}{}
	section := ""

	doc.Find("h2, h3, table.wikitable, .gridTable, .divTable").Each(func(_ int, sel *goquery.Selection) {
		if sel.Is("h2, h3") {
			section = headingText(sel)
			return
		}
		if sel.HasClass("gridTable") || sel.HasClass("divTable") {
			appendListings(&out, seen, parseGridTable(sel, section))
			return
		}
		appendListings(&out, seen, parseWikiTable(sel, section))
	})

	if out == nil {
		out = []model.TournamentListing{}
	}
	debuglog.Printf("parse.RecentTournaments count=%d", len(out))
	return out, nil
}

func headingText(sel *goquery.Selection) string {
	headline := cleanText(sel.Find(".mw-headline").First().Text())
	if headline == "" {
		clone := sel.Clone()
		clone.Find(".mw-editsection, .mw-cite-backlink").Remove()
		headline = cleanText(clone.Text())
	}
	if strings.EqualFold(headline, "Contents") {
		return ""
	}
	return headline
}

func appendListings(out *[]model.TournamentListing, seen map[string]struct{}, rows []model.TournamentListing) {
	for _, row := range rows {
		key := strings.ToLower(row.Link)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		*out = append(*out, row)
	}
}

func parseWikiTable(table *goquery.Selection, section string) []model.TournamentListing {
	var headers []string
	var out []model.TournamentListing
	table.Find("tr").Each(func(_ int, row *goquery.Selection) {
		ths := row.ChildrenFiltered("th")
		if ths.Length() > 0 && len(headers) == 0 {
			ths.Each(func(_ int, th *goquery.Selection) {
				headers = append(headers, strings.ToLower(cleanText(th.Text())))
			})
			return
		}
		tds := row.ChildrenFiltered("td")
		if tds.Length() == 0 {
			return
		}
		item, ok := listingFromCells(tds, headers, section)
		if ok {
			out = append(out, item)
		}
	})
	return out
}

func parseGridTable(table *goquery.Selection, section string) []model.TournamentListing {
	var headers []string
	var out []model.TournamentListing
	table.Find(".gridRow, .divRow").Each(func(_ int, row *goquery.Selection) {
		if row.HasClass("gridHeader") || row.HasClass("gridTitle") || row.HasClass("sortbottom") {
			if row.HasClass("gridHeader") && len(headers) == 0 {
				row.ChildrenFiltered(".gridCell, .divCell").Each(func(_ int, cell *goquery.Selection) {
					headers = append(headers, strings.ToLower(cleanText(cell.Text())))
				})
			}
			return
		}
		cells := row.ChildrenFiltered(".gridCell, .divCell")
		if cells.Length() == 0 {
			return
		}
		item, ok := listingFromGridCells(cells, headers, section)
		if ok {
			out = append(out, item)
		}
	})
	return out
}

func listingFromGridCells(cells *goquery.Selection, headers []string, section string) (model.TournamentListing, bool) {
	var (
		link, name string
		dates      *goquery.Selection
		tier       *goquery.Selection
	)
	cells.Each(func(i int, cell *goquery.Selection) {
		class := strings.ToLower(cell.AttrOr("class", ""))
		switch {
		case strings.Contains(class, "tournament") || strings.Contains(class, "event") || classHasName(class):
			if link == "" {
				link, name, _ = tournamentLinkFromCell(cell)
			}
		case strings.Contains(class, "date"):
			dates = cell
		case strings.Contains(class, "tier"):
			tier = cell
		}
		if link == "" && headerMatches(headers, i, "tournament", "event", "name") {
			link, name, _ = tournamentLinkFromCell(cell)
		}
		if dates == nil && headerMatches(headers, i, "date", "dates") {
			dates = cell
		}
		if tier == nil && headerMatches(headers, i, "tier") {
			tier = cell
		}
	})
	if link == "" {
		cells.EachWithBreak(func(_ int, cell *goquery.Selection) bool {
			l, n, ok := tournamentLinkFromCell(cell)
			if !ok {
				return true
			}
			link, name = l, n
			return false
		})
	}
	return finishListing(link, name, dates, tier, section)
}

func listingFromCells(tds *goquery.Selection, headers []string, section string) (model.TournamentListing, bool) {
	var (
		link, name string
		dates      *goquery.Selection
		tier       *goquery.Selection
	)
	tds.Each(func(i int, cell *goquery.Selection) {
		if headerMatches(headers, i, "tournament", "event", "name") {
			if link == "" {
				link, name, _ = tournamentLinkFromCell(cell)
			}
		}
		if headerMatches(headers, i, "date", "dates") {
			dates = cell
		}
		if headerMatches(headers, i, "tier") {
			tier = cell
		}
	})
	if link == "" {
		tds.EachWithBreak(func(_ int, cell *goquery.Selection) bool {
			l, n, ok := tournamentLinkFromCell(cell)
			if !ok {
				return true
			}
			link, name = l, n
			return false
		})
	}
	return finishListing(link, name, dates, tier, section)
}

func finishListing(link, name string, dates, tier *goquery.Selection, section string) (model.TournamentListing, bool) {
	if link == "" || name == "" {
		return model.TournamentListing{}, false
	}
	item := model.TournamentListing{Link: link, Name: name}
	if dates != nil {
		item.StartDate, item.EndDate = parseListingDates(cleanText(dates.Text()))
	}
	if tier != nil {
		if t := listingTierText(tier); t != "" {
			item.Tier = &t
		}
	}
	if section != "" {
		s := section
		item.Section = &s
	}
	return item, true
}

func classHasName(class string) bool {
	for _, part := range strings.Fields(class) {
		if part == "name" {
			return true
		}
	}
	return false
}

func headerMatches(headers []string, i int, names ...string) bool {
	if i < 0 || i >= len(headers) {
		return false
	}
	h := headers[i]
	for _, n := range names {
		if strings.Contains(h, n) {
			return true
		}
	}
	return false
}

func tournamentLinkFromCell(cell *goquery.Selection) (link, name string, ok bool) {
	cell.Find("a[href]").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		if a.Closest(".league-icon-small-image, .sprite-image").Length() > 0 {
			return true
		}
		n := cleanText(a.Text())
		if n == "" {
			return true
		}
		href, _ := a.Attr("href")
		u := profileURL(href)
		if u == nil {
			return true
		}
		ref, err := liquipedia.ParsePageRef(*u)
		if err != nil || skipListedTitle(ref.Title) || isRaceProfileLink(*u) {
			return true
		}
		link, name, ok = *u, n, true
		return false
	})
	return link, name, ok
}

func skipListedTitle(title string) bool {
	t := strings.ToLower(strings.TrimSpace(title))
	if t == "" || t == "main_page" {
		return true
	}
	for _, p := range []string{
		"template:", "category:", "file:", "special:", "user:",
		"help:", "liquipedia:", "mediawiki:", "module:", "portal:",
		"talk:",
	} {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	switch t {
	case "leagues/recent_tournaments", "leagues", "recent_tournaments":
		return true
	}
	if !strings.Contains(t, "/") && strings.HasSuffix(t, "_tournaments") {
		return true
	}
	return false
}

func listingTierText(cell *goquery.Selection) string {
	t := cleanText(cell.Find("a").First().Text())
	if t == "" {
		t = cleanText(cell.Text())
	}
	t = strings.TrimSpace(strings.TrimSuffix(t, "Tournaments"))
	if t == "" || t == "?" {
		return ""
	}
	return t
}

func parseListingDates(text string) (start, end *string) {
	dates := isoDateRE.FindAllString(text, 2)
	if len(dates) >= 1 {
		s := dates[0]
		start = &s
	}
	if len(dates) >= 2 {
		e := dates[1]
		end = &e
	}
	return start, end
}
