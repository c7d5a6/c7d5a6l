package model

// PlayerPage is the parsed representation of a Liquipedia player page.
// Link is required; every other field is optional until parsing fills it in.
// PortraitURL is the Liquipedia source image URL (not for browser hotlinking).
// HasPortrait is true when a cached portrait blob exists in the database.
type PlayerPage struct {
	Link          string    `json:"link"`
	Name          *string   `json:"name"`
	RealName      *string   `json:"realName"`
	IDs           []string  `json:"ids"`
	PreferredRace *string   `json:"preferredRace"`
	PortraitURL   *string   `json:"portraitUrl"`
	HasPortrait   bool      `json:"hasPortrait"`
	RaceElos      []RaceElo `json:"raceElos"`
}

// RaceElo is a player's rating for one race.
type RaceElo struct {
	Race string  `json:"race"`
	Elo  float64 `json:"elo"`
}

// NewPlayerPage returns an empty player shell for a validated page link.
func NewPlayerPage(link string) PlayerPage {
	return PlayerPage{
		Link:     link,
		IDs:      []string{},
		RaceElos: []RaceElo{},
	}
}
