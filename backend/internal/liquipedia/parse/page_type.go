package parse

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
)

// PageType is the kind of Liquipedia page inferred from the main infobox.
type PageType string

const (
	PageTypeTournament PageType = "tournament"
	PageTypePlayer     PageType = "player"
	PageTypeUnknown    PageType = "unknown"
)

// PageTypeFromHTML classifies a Liquipedia HTML fragment.
func PageTypeFromHTML(html string) (PageType, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return PageTypeUnknown, fmt.Errorf("parse html: %w", err)
	}
	pt := DetectPageType(doc)
	debuglog.Printf("DetectPageType=%s", pt)
	return pt, nil
}

// DetectPageType classifies a page from its primary infobox.
// Prefers data-analytics-infobox-type; falls back to Infobox template links.
func DetectPageType(doc *goquery.Document) PageType {
	if raw, ok := doc.Find(".fo-nttax-infobox-container").First().Attr("data-analytics-infobox-type"); ok {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "tournament":
			return PageTypeTournament
		case "person":
			return PageTypePlayer
		}
		return PageTypeUnknown
	}

	href, ok := doc.Find(`.fo-nttax-infobox a[href*="Template:Infobox_"]`).First().Attr("href")
	if !ok {
		title, tok := doc.Find(`.fo-nttax-infobox a[title^="Template:Infobox"]`).First().Attr("title")
		if tok {
			return pageTypeFromInfoboxTemplate(title)
		}
		return PageTypeUnknown
	}
	return pageTypeFromInfoboxTemplate(href)
}

func pageTypeFromInfoboxTemplate(s string) PageType {
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "infobox_league"), strings.Contains(lower, "infobox league"):
		return PageTypeTournament
	case strings.Contains(lower, "infobox_player"), strings.Contains(lower, "infobox player"):
		return PageTypePlayer
	default:
		return PageTypeUnknown
	}
}
