package parse

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// Tournament parses a Liquipedia tournament HTML page into the domain model.
// Each field is filled by its own dedicated parser method.
func Tournament(link string, html string) (model.TournamentPage, error) {
	debuglog.Printf("parse.Tournament link=%s htmlBytes=%d", link, len(html))
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
	debuglog.Printf("parse.Tournament name=%s", debuglog.Str(name))

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
	debuglog.Printf("parse.Tournament participants=%d", len(participants))

	results, err := Results(doc)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse results: %w", err)
	}
	page.Results = results
	debuglog.Printf("parse.Tournament results=%d", len(results))

	groups, err := Groups(doc, results)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse groups: %w", err)
	}
	page.Groups = groups
	debuglog.Printf("parse.Tournament groups=%d", len(groups))

	finished, err := Finished(doc)
	if err != nil {
		return model.TournamentPage{}, fmt.Errorf("parse finished: %w", err)
	}
	page.Finished = finished
	debuglog.Printf("parse.Tournament done finished=%s", debuglog.Bool(finished))

	return page, nil
}
