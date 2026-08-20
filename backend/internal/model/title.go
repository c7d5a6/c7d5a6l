package model

// Title kinds.
const (
	TitleKindFantasy    = "fantasy"
	TitleKindTournament = "tournament"
)

// UserTitle is an award on a user (fantasy star or tournament cup).
type UserTitle struct {
	ID                int64   `json:"id"`
	UserID            int64   `json:"userId"`
	UserAlias         string  `json:"userAlias"`
	Kind              string  `json:"kind"`
	Name              string  `json:"name"`
	FantasyLeagueID   *int64  `json:"fantasyLeagueId"`
	FantasyLeagueName *string `json:"fantasyLeagueName,omitempty"`
	HasImage          bool    `json:"hasImage"`
	Date              *string `json:"date"`
	CreatedAt         string  `json:"createdAt"`
}
