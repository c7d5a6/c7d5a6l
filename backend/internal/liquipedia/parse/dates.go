package parse

import "github.com/PuerkitoBio/goquery"

// resolveTournamentDates reads start/end from the infobox, falling back to a single Date row.
func resolveTournamentDates(doc *goquery.Document) (start, end *string, err error) {
	start, err = infoboxValue(doc, "Start Date")
	if err != nil {
		return nil, nil, err
	}
	end, err = infoboxValue(doc, "End Date")
	if err != nil {
		return nil, nil, err
	}
	if start != nil && end != nil {
		return start, end, nil
	}

	dateRaw, err := infoboxValue(doc, "Date")
	if err != nil {
		return nil, nil, err
	}
	if dateRaw == nil {
		return start, end, nil
	}

	fbStart, fbEnd := parseListingDates(*dateRaw)
	if start == nil {
		start = fbStart
	}
	if end == nil {
		end = fbEnd
		if end == nil {
			end = fbStart
		}
	}
	return start, end, nil
}
