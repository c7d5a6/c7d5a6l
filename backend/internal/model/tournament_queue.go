package model

// TournamentListing is one row parsed from Liquipedia Recent Tournaments.
type TournamentListing struct {
	Link      string  `json:"link"`
	Name      string  `json:"name"`
	StartDate *string `json:"startDate,omitempty"`
	EndDate   *string `json:"endDate,omitempty"`
	Tier      *string `json:"tier,omitempty"`
	Section   *string `json:"section,omitempty"`
}

// Admin tournament list filters (query tab=).
const (
	AdminFilterAll      = "all"
	AdminFilterQueue    = "queue"
	AdminFilterOngoing  = "ongoing"
	AdminFilterParsed   = "parsed"
	AdminFilterFinished = "finished"
	AdminFilterIgnored  = "ignored"
	AdminFilterFantasy  = "fantasy"
)

// AdminTournament is one row on the admin tournaments page.
type AdminTournament struct {
	QueueID         *int64   `json:"queueId"`
	TournamentID    *int64   `json:"tournamentId"`
	Link            string   `json:"link"`
	Name            *string  `json:"name"`
	StartDate       *string  `json:"startDate"`
	EndDate         *string  `json:"endDate"`
	LiquipediaTier  *string  `json:"liquipediaTier"`
	Section         *string  `json:"section,omitempty"`
	Disabled        bool     `json:"disabled"`
	Finished        *bool    `json:"finished"`
	FantasyLeagueID *int64   `json:"fantasyLeagueId"`
	Flags           []string `json:"flags"`
}

// AdminTournamentList is a paginated admin tournament listing.
type AdminTournamentList struct {
	Items    []AdminTournament `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}
