package model

// TournamentPage is the parsed representation of a Liquipedia tournament page.
// Link is required; every other field is optional until parsing fills it in.
type TournamentPage struct {
	Link           string            `json:"link"`
	Name           *string           `json:"name"`
	StartDate      *string           `json:"startDate"`
	EndDate        *string           `json:"endDate"`
	LiquipediaTier *string           `json:"liquipediaTier"`
	PlayerCounts   *PlayerCounts     `json:"playerCounts"`
	Participants   []Participant     `json:"participants"`
	Results        []Result          `json:"results"`
	Groups         []TournamentGroup `json:"groups"`
	Finished       *bool             `json:"finished"`
}

// TournamentGroup is a named pool of players within a tournament phase.
type TournamentGroup struct {
	ID        int64         `json:"id,omitempty"`
	Name      string        `json:"name"`
	Phase     string        `json:"phase"`
	SortOrder int           `json:"sortOrder"`
	Players   []Participant `json:"players"`
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
	Name     *string `json:"name"`
	Link     *string `json:"link"`
	Race     *string `json:"race"`
	Excluded bool    `json:"excluded"`
	// IsWinner is set for group standings members who advanced / topped the group.
	IsWinner bool `json:"isWinner"`
}

// Result is a scheduled or completed match between two sides.
type Result struct {
	Played       bool         `json:"played"`
	ScoreA       *int         `json:"scoreA"`
	ScoreB       *int         `json:"scoreB"`
	ParticipantA *Participant `json:"participantA"`
	ParticipantB *Participant `json:"participantB"`
	DateTime     *string      `json:"dateTime"`
	Stage        *string      `json:"stage"`
	Phase        string       `json:"phase"`
	Round        string       `json:"round"`
	GroupID      *int64       `json:"groupId,omitempty"`
	Order        int          `json:"order"`
}

// NewTournamentPage returns an empty tournament shell for a validated page link.
func NewTournamentPage(link string) TournamentPage {
	return TournamentPage{
		Link:         link,
		Participants: []Participant{},
		Results:      []Result{},
		Groups:       []TournamentGroup{},
	}
}

// TournamentSummary is a lightweight tournament row for pickers.
type TournamentSummary struct {
	ID   int64   `json:"id"`
	Link string  `json:"link"`
	Name *string `json:"name"`
}
