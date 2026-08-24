package model

// FantasyLeague is a fantasy competition bound to one tournament.
type FantasyLeague struct {
	ID             int64   `json:"id"`
	TournamentID   int64   `json:"tournamentId"`
	TournamentLink string  `json:"tournamentLink"`
	TournamentName *string `json:"tournamentName"`
	Started        bool    `json:"started"`
	Finished       bool    `json:"finished"`
	MaxPlayers     int     `json:"maxPlayers"`
	MaxCost        int     `json:"maxCost"`
}

// FantasyPlayerRow is a fantasy_player joined with tournament/player display fields.
type FantasyPlayerRow struct {
	ID                 int64    `json:"id"`
	FantasyLeagueID    int64    `json:"fantasyLeagueId"`
	TournamentPlayerID int64    `json:"tournamentPlayerId"`
	Name               *string  `json:"name"`
	Link               *string  `json:"link"`
	Race               *string  `json:"race"`
	Cost               int      `json:"cost"`
	PointsRo24         *int     `json:"pointsRo24"`
	PointsRo16         *int     `json:"pointsRo16"`
	PointsRo8          *int     `json:"pointsRo8"`
	PointsRo4          *int     `json:"pointsRo4"`
	PointsRo2          *int     `json:"pointsRo2"`
	PointsEarned       int      `json:"pointsEarned"`
	Defeated           bool     `json:"defeated"`
	IsWinner           bool     `json:"isWinner"`
	Elo                *float64 `json:"elo,omitempty"`
}

// FantasyPreviewPlayer is a tournament roster row with suggested fantasy cost.
type FantasyPreviewPlayer struct {
	TournamentPlayerID int64   `json:"tournamentPlayerId"`
	Name               *string `json:"name"`
	Link               *string `json:"link"`
	Race               *string `json:"race"`
	Elo                float64 `json:"elo"`
	Cost               int     `json:"cost"`
}

// FantasyTeamMemberRow is one roster slot on a fantasy team.
type FantasyTeamMemberRow struct {
	FantasyPlayerID int64   `json:"fantasyPlayerId"`
	Name            *string `json:"name"`
	Link            *string `json:"link"`
	Race            *string `json:"race"`
	Cost            int     `json:"cost"`
	PointsEarned    int     `json:"pointsEarned"`
	Defeated        bool    `json:"defeated"`
	IsWinner        bool    `json:"isWinner"`
	Elo             float64 `json:"elo"`
}

// FantasyTeamRow is a fantasy team with owner alias and members.
type FantasyTeamRow struct {
	ID              int64                  `json:"id"`
	FantasyLeagueID int64                  `json:"fantasyLeagueId"`
	UserID          int64                  `json:"userId"`
	UserAlias       string                 `json:"userAlias"`
	Rank            int                    `json:"rank"`
	Points          int                    `json:"points"`
	Cost            int                    `json:"cost"`
	Members         []FantasyTeamMemberRow `json:"members"`
	Titles          []UserTitle            `json:"titles"`
}

// FantasyPlayerCostOverride sets cost for a tournament player at create time.
type FantasyPlayerCostOverride struct {
	TournamentPlayerID int64 `json:"tournamentPlayerId"`
	Cost               int   `json:"cost"`
}

// FantasyGroupPlayer is a tournament group member with fantasy cost (no points).
type FantasyGroupPlayer struct {
	FantasyPlayerID int64   `json:"fantasyPlayerId"`
	Name            *string `json:"name"`
	Link            *string `json:"link"`
	Race            *string `json:"race"`
	Cost            int     `json:"cost"`
	Excluded        bool    `json:"excluded"`
	// IsGroupWinner marks standings advancement / group top (not fantasy champion).
	IsGroupWinner bool `json:"isGroupWinner"`
}

// FantasyGroup is a tournament group with fantasy costs for drafting.
type FantasyGroup struct {
	ID        int64                `json:"id"`
	Name      string               `json:"name"`
	Phase     string               `json:"phase"`
	SortOrder int                  `json:"sortOrder"`
	Players   []FantasyGroupPlayer `json:"players"`
}

// FantasyMatchBoard is groups + results for the Results tab / today panel.
type FantasyMatchBoard struct {
	Groups  []FantasyGroup `json:"groups"`
	Results []Result       `json:"results"`
	Today   string         `json:"today"` // YYYY-MM-DD UTC
}
