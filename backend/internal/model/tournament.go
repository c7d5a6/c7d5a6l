package model

// TournamentPage is the parsed representation of a Liquipedia tournament page.
// Link is required; every other field is optional until parsing fills it in.
type TournamentPage struct {
	Link           string         `json:"link"`
	Name           *string        `json:"name"`
	StartDate      *string        `json:"startDate"`
	EndDate        *string        `json:"endDate"`
	LiquipediaTier *string        `json:"liquipediaTier"`
	PlayerCounts   *PlayerCounts  `json:"playerCounts"`
	Participants   []Participant  `json:"participants"`
	Results        []Result       `json:"results"`
	Finished       *bool          `json:"finished"`
}

// PlayerCounts breaks down entrants by race. All counts are optional.
type PlayerCounts struct {
	Total   *int `json:"total"`
	Protoss *int `json:"protoss"`
	Zerg    *int `json:"zerg"`
	Terran  *int `json:"terran"`
}

// Participant is a tournament entrant.
type Participant struct {
	Name *string `json:"name"`
}

// Result will hold placement / match outcome data. Empty for now.
type Result struct{}

// NewTournamentPage returns an empty tournament shell for a validated page link.
func NewTournamentPage(link string) TournamentPage {
	return TournamentPage{
		Link:         link,
		Participants: []Participant{},
		Results:      []Result{},
	}
}
