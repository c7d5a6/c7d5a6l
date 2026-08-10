package parse

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Name extracts the tournament name from a Liquipedia tournament page document.
// It reads the first infobox header, ignoring edit-button chrome.
func Name(doc *goquery.Document) (*string, error) {
	header := doc.Find(".fo-nttax-infobox .infobox-header").First()
	if header.Length() == 0 {
		header = doc.Find(".infobox-header").First()
	}
	if header.Length() == 0 {
		return nil, nil
	}

	clone := header.Clone()
	clone.Find(".infobox-buttons, .navigation-not-searchable").Remove()
	name := cleanText(clone.Text())
	if name == "" {
		return nil, nil
	}
	return &name, nil
}

func cleanText(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
