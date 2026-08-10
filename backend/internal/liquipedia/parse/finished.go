package parse

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Finished reports whether the tournament has a determined first-place winner.
// Liquipedia marks unfinished prize pools with "TBD" in the 1st-place slot.
func Finished(doc *goquery.Document) (*bool, error) {
	badge := doc.Find(".prizepooltable .placement-1").First()
	if badge.Length() == 0 {
		return nil, nil
	}

	row := badge.Closest("tr")
	if row.Length() == 0 {
		return nil, nil
	}

	name := cleanText(row.Find("td.prizepooltable-col-team .name").First().Text())
	if name == "" {
		name = cleanText(row.Find("td.prizepooltable-col-team").First().Text())
	}
	if name == "" {
		return nil, nil
	}

	finished := !strings.EqualFold(name, "TBD")
	return &finished, nil
}
