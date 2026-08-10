package parse

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// infoboxValue returns the content cell next to an infobox description label.
func infoboxValue(doc *goquery.Document, label string) (*string, error) {
	want := normalizeLabel(label)
	var found *string

	doc.Find(".fo-nttax-infobox .infobox-description").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		if normalizeLabel(sel.Text()) != want {
			return true
		}
		content := sel.Next()
		if content.Length() == 0 {
			content = sel.Parent().Children().Eq(1)
		}
		if content.Length() == 0 {
			return false
		}
		text := cleanText(content.Text())
		if text == "" {
			return false
		}
		found = &text
		return false
	})

	return found, nil
}

func normalizeLabel(s string) string {
	s = cleanText(s)
	s = strings.TrimSuffix(s, ":")
	return strings.ToLower(s)
}
