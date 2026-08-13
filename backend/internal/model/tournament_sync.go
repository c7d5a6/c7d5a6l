package model

// TournamentSync describes how a parsed tournament compares to the database.
type TournamentSync struct {
	Exists  bool                     `json:"exists"`
	Same    bool                     `json:"same"`
	Action  string                   `json:"action"` // "add" | "update" | "none"
	Stored  *TournamentPage          `json:"stored,omitempty"`
	Changes []FieldChange            `json:"changes,omitempty"`
	Players []TournamentPlayerStatus `json:"players,omitempty"`
}

// TournamentPlayerStatus is one participant's DB presence for import planning.
type TournamentPlayerStatus struct {
	Name       *string `json:"name"`
	Link       *string `json:"link"`
	Race       *string `json:"race"`
	Excluded   bool    `json:"excluded"`
	InDatabase bool    `json:"inDatabase"`
	WillImport bool    `json:"willImport"`
	SkipReason *string `json:"skipReason,omitempty"`
}

const (
	TournamentActionAdd    = "add"
	TournamentActionUpdate = "update"
	TournamentActionNone   = "none"
)
