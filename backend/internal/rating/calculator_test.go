package rating_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/rating"
)

func TestCalculator_identityStub(t *testing.T) {
	start := map[int64]float64{1: 1750, 2: 1800}
	matches := []rating.Match{
		{PlayerRaceA: 1, PlayerRaceB: 2, ScoreA: 2, ScoreB: 1, Played: true},
	}
	var calc rating.Calculator
	got := calc.Compute(start, matches, nil)
	if got[1] != 1750 || got[2] != 1800 {
		t.Fatalf("identity stub changed elos: %#v", got)
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
