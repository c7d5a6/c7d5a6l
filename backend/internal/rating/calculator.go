package rating

import "math"

const (
	seasonMatchK         = 85.0
	flInactiveDefaultElo = 1900.0
	flInactivePenalty    = 60.0
	flMapPoints          = 400.0
)

// Match is one played tournament_result row for rating calculation.
type Match struct {
	PlayerRaceA int64
	PlayerRaceB int64
	ScoreA      int
	ScoreB      int
	Played      bool
}

// Calculator applies rating formulas.
type Calculator struct{}

// Compute recalculates elos from start ratings using season tournament matches,
// then applies an additional pass with fantasy-league tournament matches.
func (c *Calculator) Compute(start map[int64]float64, seasonMatches, flMatches []Match) map[int64]float64 {
	out := copyElos(start)
	seasonPlayed := make(map[int64]bool)
	c.applySeasonMatches(out, seasonMatches, seasonPlayed)
	c.applyFantasyLeagueMatches(out, flMatches, seasonPlayed)
	return out
}

// applySeasonMatches is pass 1 — one update per match using map scores as fractional result.
func (c *Calculator) applySeasonMatches(elos map[int64]float64, matches []Match, seasonPlayed map[int64]bool) {
	for _, m := range matches {
		if !m.Played || m.ScoreA < 0 || m.ScoreB < 0 {
			continue
		}
		w1, w2 := float64(m.ScoreA), float64(m.ScoreB)
		if w1+w2 == 0 {
			continue
		}
		r1, ok1 := elos[m.PlayerRaceA]
		r2, ok2 := elos[m.PlayerRaceB]
		if !ok1 || !ok2 {
			continue
		}
		seasonPlayed[m.PlayerRaceA] = true
		seasonPlayed[m.PlayerRaceB] = true

		actual1 := w1 / (w1 + w2)
		actual2 := w2 / (w1 + w2)
		p1 := math.Pow(10, r1/400)
		p2 := math.Pow(10, r2/400)
		expected1 := p1 / (p1 + p2)
		expected2 := p2 / (p1 + p2)

		elos[m.PlayerRaceA] = r1 + seasonMatchK*(actual1-expected1)
		elos[m.PlayerRaceB] = r2 + seasonMatchK*(actual2-expected2)
	}
}

// applyFantasyLeagueMatches is pass 2 — FL tournament.
// Type 1 (FL maps): aggregated FL update.
// Type 2 (season matches, no FL): unchanged after pass 1.
// Type 3 (zero season matches): inactive penalty.
func (c *Calculator) applyFantasyLeagueMatches(elos map[int64]float64, matches []Match, seasonPlayed map[int64]bool) {
	stats := make(map[int64]*flMapStats)
	hadMaps := false

	for _, m := range matches {
		if !m.Played || m.ScoreA < 0 || m.ScoreB < 0 {
			continue
		}
		if m.ScoreA == 0 && m.ScoreB == 0 {
			continue
		}
		hadMaps = true
		for range m.ScoreA {
			recordFLMap(stats, m.PlayerRaceA, m.PlayerRaceB)
		}
		for range m.ScoreB {
			recordFLMap(stats, m.PlayerRaceB, m.PlayerRaceA)
		}
	}

	if !hadMaps {
		return
	}

	flStart := copyElos(elos)

	for id, st := range stats {
		pMaps := st.mapWins + st.mapLosses
		if pMaps == 0 {
			continue
		}
		var oppSum float64
		for oppID, n := range st.mapsVsOpp {
			oppRating := flStart[oppID]
			if oppRating == 0 {
				oppRating = flInactiveDefaultElo
			}
			oppSum += oppRating * float64(n)
		}
		pSumm := flMapPoints*float64(st.mapWins-st.mapLosses) + oppSum
		perfElo := pSumm / float64(pMaps)
		elos[id] = flPlayedRating(flStart[id], perfElo)
	}

	for id, pS := range flStart {
		if st, ok := stats[id]; ok && st.mapWins+st.mapLosses > 0 {
			continue
		}
		if seasonPlayed[id] {
			continue
		}
		elos[id] = flInactiveRating(pS)
	}
}

type flMapStats struct {
	mapWins, mapLosses int
	mapsVsOpp          map[int64]int
}

func recordFLMap(stats map[int64]*flMapStats, winnerID, loserID int64) {
	win := statsForFL(stats, winnerID)
	lose := statsForFL(stats, loserID)
	win.mapWins++
	win.mapsVsOpp[loserID]++
	lose.mapLosses++
	lose.mapsVsOpp[winnerID]++
}

func statsForFL(stats map[int64]*flMapStats, id int64) *flMapStats {
	st, ok := stats[id]
	if !ok {
		st = &flMapStats{mapsVsOpp: make(map[int64]int)}
		stats[id] = st
	}
	return st
}

// flPlayedRating: p1r = p1s*e1 + elo*(1-e1).
func flPlayedRating(p1s, perfElo float64) float64 {
	e1 := flExpectedScore(p1s, perfElo)
	return p1s*e1 + perfElo*(1-e1)
}

// flInactiveRating: no FL maps — elo=1900 for e1, p1r = p1s + 60*(0-e1).
func flInactiveRating(pS float64) float64 {
	e1 := flExpectedScore(pS, flInactiveDefaultElo)
	return pS - flInactivePenalty*e1
}

func flExpectedScore(rating, oppRating float64) float64 {
	r1 := math.Pow(10, rating/400)
	r2 := math.Pow(10, oppRating/400)
	return r1 / (r1 + r2)
}

func copyElos(start map[int64]float64) map[int64]float64 {
	out := make(map[int64]float64, len(start))
	for id, elo := range start {
		out[id] = elo
	}
	return out
}
