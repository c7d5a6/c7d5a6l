package rating

// Match is one played tournament_result row for rating calculation.
type Match struct {
	PlayerRaceA int64
	PlayerRaceB int64
	ScoreA      int
	ScoreB      int
	Played      bool
}

// Calculator applies rating formulas. Edit Compute and applyMatches to implement formulas.
type Calculator struct{}

// Compute recalculates elos from start ratings using season tournament matches,
// then applies an additional pass with fantasy-league tournament matches.
func (c *Calculator) Compute(start map[int64]float64, seasonMatches, flMatches []Match) map[int64]float64 {
	out := copyElos(start)
	c.applyMatches(out, seasonMatches)
	c.applyMatches(out, flMatches)
	return out
}

func (c *Calculator) applyMatches(elos map[int64]float64, matches []Match) {
	for _, m := range matches {
		if !m.Played {
			continue
		}
		_ = elos[m.PlayerRaceA]
		_ = elos[m.PlayerRaceB]
		// TODO: implement per-match rating update using elos[m.PlayerRaceA], elos[m.PlayerRaceB], scores.
	}
}

func copyElos(start map[int64]float64) map[int64]float64 {
	out := make(map[int64]float64, len(start))
	for id, elo := range start {
		out[id] = elo
	}
	return out
}
