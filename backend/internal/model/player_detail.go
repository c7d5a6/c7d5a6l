package model

// PlayerDetail is the dossier payload for a single player page.
type PlayerDetail struct {
	ID            int64             `json:"id"`
	Link          string            `json:"link"`
	Name          *string           `json:"name"`
	RealName      *string           `json:"realName"`
	IDs           []string          `json:"ids"`
	PreferredRace *string           `json:"preferredRace"`
	HasPortrait   bool              `json:"hasPortrait"`
	Races         []PlayerRaceEntry `json:"races"`
	Winrates      PlayerWinrates    `json:"winrates"`
	SeasonMatches []PlayerMatch     `json:"seasonMatches"`
	Season        *SeasonSummary    `json:"season,omitempty"`
}

// PlayerWinrates holds map-based winrate blocks for each scope.
type PlayerWinrates struct {
	Season     WinrateBlock `json:"season"`
	Fantasy    WinrateBlock `json:"fantasy"`
	Overall    WinrateBlock `json:"overall"`
	AllFantasy WinrateBlock `json:"allFantasy"`
}

// WinrateBlock is maps won/lost vs each race and overall.
type WinrateBlock struct {
	VsAll     WinrateStat `json:"vsAll"`
	VsTerran  WinrateStat `json:"vsTerran"`
	VsZerg    WinrateStat `json:"vsZerg"`
	VsProtoss WinrateStat `json:"vsProtoss"`
}

// WinrateStat is map wins/losses and percentage (0–100).
// Rate is nil when there are no maps yet.
type WinrateStat struct {
	Wins   int      `json:"wins"`
	Losses int      `json:"losses"`
	Rate   *float64 `json:"rate"`
}

// PlayerMatch is one played series from the player's perspective.
type PlayerMatch struct {
	ID               int64   `json:"id"`
	TournamentID     int64   `json:"tournamentId"`
	TournamentLink   string  `json:"tournamentLink"`
	TournamentName   *string `json:"tournamentName"`
	Played           bool    `json:"played"`
	ScoreMine        int     `json:"scoreMine"`
	ScoreOpp         int     `json:"scoreOpp"`
	MyRace           string  `json:"myRace"`
	OpponentName     *string `json:"opponentName"`
	OpponentLink     *string `json:"opponentLink"`
	OpponentRace     *string `json:"opponentRace"`
	OpponentPlayerID *int64  `json:"opponentPlayerId,omitempty"`
	Phase            string  `json:"phase"`
	Round            string  `json:"round"`
	DateTime         *string `json:"dateTime"`
}
