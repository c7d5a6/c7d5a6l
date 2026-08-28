package rating_test

import (
	"math"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/rating"
)

func TestCalculator_seasonMatchUpdates(t *testing.T) {
	start := map[int64]float64{1: 1750, 2: 1800}
	matches := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 2, ScoreB: 1, Played: true},
	}
	var calc rating.Calculator
	got := calc.Compute(start, matches, nil)

	w1, w2 := 2.0, 1.0
	r1, r2 := 1750.0, 1800.0
	p1 := math.Pow(10, r1/400)
	p2 := math.Pow(10, r2/400)
	want1 := r1 + 85*(w1/(w1+w2)-p1/(p1+p2))
	want2 := r2 + 85*(w2/(w1+w2)-p2/(p1+p2))

	if math.Abs(got[1]-want1) > 0.01 {
		t.Fatalf("player 1: got=%v want=%v", got[1], want1)
	}
	if math.Abs(got[2]-want2) > 0.01 {
		t.Fatalf("player 2: got=%v want=%v", got[2], want2)
	}
}

func TestCalculator_seasonMatchGolden(t *testing.T) {
	const (
		r1 = 2661.58072343516
		r2 = 3587.15554287436
	)
	start := map[int64]float64{1: r1, 2: r2}
	matches := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 4, ScoreB: 3, Played: true},
	}
	var calc rating.Calculator
	got := calc.Compute(start, matches, nil)

	if round1(got[1]) != 2709.7 {
		t.Fatalf("player 1: got=%v want=2709.7", round1(got[1]))
	}
	if round1(got[2]) != 3539.0 {
		t.Fatalf("player 2: got=%v want=3539.0", round1(got[2]))
	}
}

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}

func TestCalculator_flUsesPostSeasonOpponentRatings(t *testing.T) {
	start := map[int64]float64{1: 1750, 2: 1800}
	season := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 2, ScoreB: 1, Played: true},
	}
	fl := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 0, ScoreB: 1, Played: true},
	}
	var calc rating.Calculator

	flOnly := calc.Compute(start, nil, fl)
	both := calc.Compute(start, season, fl)

	// Same FL map loss, but opponent rating in pSumm differs after season pass.
	if flOnly[1] == both[1] {
		t.Fatalf("FL should use post-season opponent rating: flOnly=%v both=%v", flOnly[1], both[1])
	}

	// Verify against manual: season first, then FL aggregate using updated opp rating.
	afterSeason := calc.Compute(start, season, nil)
	oppAfterSeason := afterSeason[2]
	perf := (-400.0 + oppAfterSeason) / 1.0
	want := flPlayed(t, afterSeason[1], perf)
	if math.Abs(both[1]-want) > 0.01 {
		t.Fatalf("got=%v want=%v", both[1], want)
	}
}

func TestCalculator_seasonThenFL(t *testing.T) {
	start := map[int64]float64{1: 1750, 2: 1800}
	season := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 2, ScoreB: 1, Played: true},
	}
	fl := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 1, ScoreB: 0, Played: true},
	}
	var calc rating.Calculator
	seasonOnly := calc.Compute(start, season, nil)
	both := calc.Compute(start, season, fl)

	if seasonOnly[1] == start[1] {
		t.Fatal("season pass should change ratings")
	}
	if both[1] == seasonOnly[1] {
		t.Fatal("FL pass should apply on top of season pass")
	}
}

func TestCalculator_flNoPassWhenNoMatches(t *testing.T) {
	start := map[int64]float64{1: 1750, 2: 1800}
	var calc rating.Calculator
	got := calc.Compute(start, nil, nil)
	if got[1] != 1750 || got[2] != 1800 {
		t.Fatalf("empty FL pass should not change elos: %#v", got)
	}
}

func TestCalculator_seasonOnlyNoFLPenalty(t *testing.T) {
	start := map[int64]float64{1: 1750, 2: 1800, 3: 1700}
	season := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 3, ScoreA: 2, ScoreB: 1, Played: true},
	}
	fl := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 1, ScoreB: 0, Played: true},
	}
	var calc rating.Calculator

	seasonOnly := calc.Compute(start, season, nil)
	got := calc.Compute(start, season, fl)

	// Player 3: season match, no FL — regular update only, no inactive penalty.
	if got[3] != seasonOnly[3] {
		t.Fatalf("season-only player: got=%v want=%v (season-only)", got[3], seasonOnly[3])
	}
	if got[3] == start[3] {
		t.Fatal("player 3 should still get regular match update")
	}

	// Player 2: FL only, no season — FL update, not penalty (played FL maps).
	if got[2] == start[2] {
		t.Fatal("player 2 should get FL update")
	}
}

func TestCalculator_flInactiveGolden(t *testing.T) {
	const pS = 2332.1255790638
	start := map[int64]float64{1: pS, 2: 2000, 3: 2100}
	fl := []rating.Match{
		{PlayerRaceA: 2, PlayerRaceB: 3, ScoreA: 1, ScoreB: 0, Played: true},
	}
	var calc rating.Calculator
	got := calc.Compute(start, nil, fl)

	if round1(got[1]) != 2276.7 {
		t.Fatalf("inactive player: got=%v want=2276.7", round1(got[1]))
	}
}

func TestCalculator_flInactiveWhenTournamentHadMaps(t *testing.T) {
	start := map[int64]float64{1: 1750, 2: 1800, 3: 1700}
	fl := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 1, ScoreB: 0, Played: true},
	}
	var calc rating.Calculator
	got := calc.Compute(start, nil, fl)

	r1 := math.Pow(10, 1700.0/400)
	r2 := math.Pow(10, 1900.0/400)
	e1 := r1 / (r1 + r2)
	want3 := 1700 - 60*e1

	if math.Abs(got[3]-want3) > 0.01 {
		t.Fatalf("non-participant: got=%v want=%v", got[3], want3)
	}
}

func TestCalculator_flMatchGolden(t *testing.T) {
	const (
		p  = 2461.0462144295
		o1 = 2339.75355401132
		o2 = 2534.04163878423
	)
	start := map[int64]float64{1: p, 2: o1, 3: o2}
	fl := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 0, ScoreB: 1, Played: true},
		{PlayerRaceA: 1, PlayerRaceB: 3, ScoreA: 0, ScoreB: 2, Played: true},
	}
	var calc rating.Calculator
	got := calc.Compute(start, nil, fl)

	if round1(got[1]) != 2423.9 {
		t.Fatalf("player: got=%v want=2423.9", round1(got[1]))
	}
}

func TestCalculator_flMapUpdatesBothPlayers(t *testing.T) {
	start := map[int64]float64{1: 1750, 2: 1800}
	fl := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 1, ScoreB: 0, Played: true},
	}
	var calc rating.Calculator
	got := calc.Compute(start, nil, fl)

	perf1 := 400.0 + 1800.0
	want1 := flPlayed(t, 1750, perf1)
	perf2 := -400.0 + 1750.0
	want2 := flPlayed(t, 1800, perf2)

	if math.Abs(got[1]-want1) > 0.01 {
		t.Fatalf("winner: got=%v want=%v", got[1], want1)
	}
	if math.Abs(got[2]-want2) > 0.01 {
		t.Fatalf("loser: got=%v want=%v", got[2], want2)
	}
}

func flPlayed(t *testing.T, p1s, perfElo float64) float64 {
	t.Helper()
	r1 := math.Pow(10, p1s/400)
	r2 := math.Pow(10, perfElo/400)
	e1 := r1 / (r1 + r2)
	return p1s*e1 + perfElo*(1-e1)
}

func TestCalculator_flBo3ExpandsToThreeMaps(t *testing.T) {
	start := map[int64]float64{1: 1750, 2: 1750}
	var calc rating.Calculator

	oneMap := calc.Compute(start, nil, []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 1, ScoreB: 0, Played: true},
	})
	threeMaps := calc.Compute(start, nil, []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 1, ScoreB: 2, Played: true},
	})

	if oneMap[1] == threeMaps[1] {
		t.Fatal("BO3 1:2 should differ from single map 1:0")
	}
	if threeMaps[1] == start[1] {
		t.Fatal("BO3 should change ratings")
	}
}

func TestCalculator_twoPassOrder(t *testing.T) {
	start := map[int64]float64{1: 1750}
	var calc rating.Calculator
	got := calc.Compute(start, nil, nil)
	if got[1] != 1750 {
		t.Fatalf("got=%v", got[1])
	}
}
