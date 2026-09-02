package model

// PlayerRaceEntry is one player_race row joined with player identity, for roster views.
type PlayerRaceEntry struct {
	PlayerRaceID      int64    `json:"playerRaceId"`
	PlayerID          int64    `json:"playerId"`
	Link              string   `json:"link"`
	Name              *string  `json:"name"`
	RealName          *string  `json:"realName"`
	PreferredRace     *string  `json:"preferredRace"`
	HasPortrait       bool     `json:"hasPortrait"`
	Race              string   `json:"race"`
	Elo               float64  `json:"elo"`
	ProjectedElo      *float64 `json:"projectedElo,omitempty"`
	SeasonStartElo    *float64 `json:"seasonStartElo,omitempty"`
	LastSeasonEndElo  *float64 `json:"lastSeasonEndElo,omitempty"`
	LastSeasonEndRank *int     `json:"lastSeasonEndRank,omitempty"`
	RankDelta         *int     `json:"rankDelta,omitempty"`
}
