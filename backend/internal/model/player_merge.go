package model

// MergeCandidate is a suggested player to merge into a main account.
type MergeCandidate struct {
	PlayerID    int64    `json:"playerId"`
	Link        string   `json:"link"`
	Name        *string  `json:"name"`
	RealName    *string  `json:"realName"`
	Aliases     []string `json:"aliases"`
	Score       int      `json:"score"`
	MatchReason string   `json:"matchReason"`
}
