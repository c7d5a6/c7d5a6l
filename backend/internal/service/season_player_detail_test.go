package service

import (
	"math"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/repository"
)

func TestWinrateBlockFromRowsMapBased(t *testing.T) {
	rows := []repository.PlayerMatchRow{
		{
			ScoreA:      2,
			ScoreB:      1,
			PlayerAID:   1,
			PlayerARace: "zerg",
			PlayerBID:   2,
			PlayerBRace: "terran",
			Played:      true,
		},
		{
			ScoreA:      2,
			ScoreB:      0,
			PlayerAID:   3,
			PlayerARace: "protoss",
			PlayerBID:   1,
			PlayerBRace: "zerg",
			Played:      true,
		},
	}

	block := winrateBlockFromRows(1, rows)
	// Maps: 2-1 vs T, then 2-0 loss vs P → total 2W 3L
	if block.VsAll.Wins != 2 || block.VsAll.Losses != 3 {
		t.Fatalf("vsAll maps = %d-%d, want 2-3", block.VsAll.Wins, block.VsAll.Losses)
	}
	if block.VsAll.Rate == nil || math.Abs(*block.VsAll.Rate-40.0) > 0.01 {
		t.Fatalf("vsAll rate = %v, want 40", block.VsAll.Rate)
	}
	if block.VsTerran.Wins != 2 || block.VsTerran.Losses != 1 {
		t.Fatalf("vsT = %d-%d, want 2-1", block.VsTerran.Wins, block.VsTerran.Losses)
	}
	wantT := 100.0 * 2 / 3
	if block.VsTerran.Rate == nil || math.Abs(*block.VsTerran.Rate-wantT) > 0.01 {
		t.Fatalf("vsT rate = %v, want %v", block.VsTerran.Rate, wantT)
	}
	if block.VsProtoss.Wins != 0 || block.VsProtoss.Losses != 2 {
		t.Fatalf("vsP = %d-%d, want 0-2", block.VsProtoss.Wins, block.VsProtoss.Losses)
	}
	if block.VsZerg.Wins != 0 || block.VsZerg.Losses != 0 || block.VsZerg.Rate != nil {
		t.Fatalf("vsZ should be empty, got %+v", block.VsZerg)
	}
}
