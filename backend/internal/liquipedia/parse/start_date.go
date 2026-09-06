package parse

import "github.com/PuerkitoBio/goquery"

// StartDate extracts the tournament start date from the Liquipedia infobox.
func StartDate(doc *goquery.Document) (*string, error) {
	start, _, err := resolveTournamentDates(doc)
	return start, err
}
