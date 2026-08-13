package model

// PlayerSync describes how a parsed player compares to the database row (if any).
type PlayerSync struct {
	Exists  bool           `json:"exists"`
	Same    bool           `json:"same"`
	Action  string         `json:"action"` // "add" | "update" | "none"
	Stored  *PlayerPage    `json:"stored,omitempty"`
	Changes []FieldChange  `json:"changes,omitempty"`
}

// FieldChange is one differing field between parsed and stored player data.
type FieldChange struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

const (
	PlayerActionAdd    = "add"
	PlayerActionUpdate = "update"
	PlayerActionNone   = "none"
)
