package parse_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

func TestPlayerCounts_AllFixtures(t *testing.T) {
	t.Parallel()

	wantByFixture := map[string]model.PlayerCounts{
		"starcraft/ASL/20.html": {
			Total:   intPtr(28),
			Protoss: intPtr(6),
			Terran:  intPtr(11),
			Zerg:    intPtr(11),
		},
		"starcraft/AfreecaTV/StarCraft_League_Remastered/14.html": {
			Total:   intPtr(28),
			Protoss: intPtr(7),
			Terran:  intPtr(10),
			Zerg:    intPtr(11),
		},
		"starcraft/AfreecaTV/StarCraft_League_Remastered/8.html": {
			Total:   intPtr(28),
			Protoss: intPtr(9),
			Terran:  intPtr(9),
			Zerg:    intPtr(10),
		},
		"starcraft/KCM/Race_Survival/2026/1.html": {
			Total:   intPtr(31),
			Protoss: intPtr(11),
			Terran:  intPtr(10),
			Zerg:    intPtr(10),
		},
	}

	fixtures := loadFixtures(t)
	if len(fixtures) != len(wantByFixture) {
		t.Fatalf("fixture count=%d, expected expectations for %d", len(fixtures), len(wantByFixture))
	}

	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.name, func(t *testing.T) {
			t.Parallel()

			want, ok := wantByFixture[fx.name]
			if !ok {
				t.Fatalf("missing expected player counts for fixture %s", fx.name)
			}

			got, err := parse.PlayerCounts(documentFromHTML(t, fx.html))
			if err != nil {
				t.Fatalf("PlayerCounts: %v", err)
			}
			if got == nil {
				t.Fatal("PlayerCounts returned nil")
			}
			assertIntPtr(t, "total", got.Total, want.Total)
			assertIntPtr(t, "protoss", got.Protoss, want.Protoss)
			assertIntPtr(t, "terran", got.Terran, want.Terran)
			assertIntPtr(t, "zerg", got.Zerg, want.Zerg)
		})
	}
}

func intPtr(n int) *int { return &n }

func assertIntPtr(t *testing.T, field string, got, want *int) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("%s: got %v, want %v", field, got, want)
	case *got != *want:
		t.Fatalf("%s: got %d, want %d", field, *got, *want)
	}
}
