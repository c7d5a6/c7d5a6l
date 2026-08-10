package parse

import "github.com/PuerkitoBio/goquery"

// LiquipediaTier extracts the Liquipedia tier from the tournament infobox.
func LiquipediaTier(doc *goquery.Document) (*string, error) {
	return infoboxValue(doc, "Liquipedia Tier")
}
