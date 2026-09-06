package parse

import "github.com/PuerkitoBio/goquery"

// EndDate extracts the tournament end date from the Liquipedia infobox.
func EndDate(doc *goquery.Document) (*string, error) {
	_, end, err := resolveTournamentDates(doc)
	return end, err
}
