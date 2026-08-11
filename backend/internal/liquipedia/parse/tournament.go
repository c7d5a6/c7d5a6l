package parse

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// Tournament parses a Liquipedia tournament HTML page into the domain model.
// Each field is filled by its own dedicated parser method.
func Tournament(link string, html string) (model.TournamentPage, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse html: %w", err)
	}

	page := model.NewTournamentPage(link)

	name, err := Name(doc)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse name: %w", err)
	}
	page.Name = name

	startDate, err := StartDate(doc)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse start date: %w", err)
	}
	page.StartDate = startDate

	endDate, err := EndDate(doc)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse end date: %w", err)
	}
	page.EndDate = endDate

	tier, err := LiquipediaTier(doc)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse liquipedia tier: %w", err)
	}
	page.LiquipediaTier = tier

	playerCounts, err := PlayerCounts(doc)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse player counts: %w", err)
	}
	page.PlayerCounts = playerCounts

	participants, err := Participants(doc)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse participants: %w", err)
	}
	page.Participants = participants

	results, err := Results(doc)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse results: %w", err)
	}
	page.Results = results

	finished, err := Finished(doc)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse finished: %w", err)
	}
	page.Finished = finished

	return page, nil
}
