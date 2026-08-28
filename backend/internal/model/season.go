package model

// Season is one rating cycle.
type Season struct {
	ID                      int64   `json:"id"`
	Name                    string  `json:"name"`
	Status                  string  `json:"status"`
	StartedAt               string  `json:"startedAt"`
	ClosedAt                *string `json:"closedAt,omitempty"`
	ReadyToClose            bool    `json:"readyToClose"`
	ClosingFantasyLeagueID  *int64  `json:"closingFantasyLeagueId,omitempty"`
	ClosingFantasyLeagueName *string `json:"closingFantasyLeagueName,omitempty"`
}

// SeasonSummary is a compact season reference for list responses.
type SeasonSummary struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	StartedAt    string  `json:"startedAt"`
	ClosedAt     *string `json:"closedAt,omitempty"`
	ReadyToClose bool    `json:"readyToClose"`
}

// ClosePreviewTournament is a tournament eligible for season-close rating selection.
type ClosePreviewTournament struct {
	ID           int64   `json:"id"`
	Link         string  `json:"link"`
	Name         *string `json:"name"`
	StartDate    *string `json:"startDate"`
	EndDate      *string `json:"endDate"`
	Finished     bool    `json:"finished"`
	Selected     bool    `json:"selected"`
	IsFantasySource bool `json:"isFantasySource"`
}

// SeasonClosePreview is the admin season-close screen payload.
type SeasonClosePreview struct {
	Season                 Season                   `json:"season"`
	Tournaments            []ClosePreviewTournament `json:"tournaments"`
	ClosingFantasyLeagueID *int64                   `json:"closingFantasyLeagueId,omitempty"`
}
