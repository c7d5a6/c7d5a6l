package model

// PlayerPage is the parsed representation of a Liquipedia player page.
// Link is required; every other field is optional until parsing fills it in.
type PlayerPage struct {
	Link          string   `json:"link"`
	Name          *string  `json:"name"`
	RealName      *string  `json:"realName"`
	IDs           []string `json:"ids"`
	PreferredRace *string  `json:"preferredRace"`
}

// NewPlayerPage returns an empty player shell for a validated page link.
func NewPlayerPage(link string) PlayerPage {
	return PlayerPage{
		Link: link,
		IDs:  []string{},
	}
}
