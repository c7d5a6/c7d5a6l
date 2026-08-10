package parse_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
)

func TestStartDate_AllFixtures(t *testing.T) {
	t.Parallel()

	wantByFixture := map[string]string{
		"starcraft/ASL/20.html": "2025-08-18",
		"starcraft/AfreecaTV/StarCraft_League_Remastered/14.html": "2022-08-09",
		"starcraft/AfreecaTV/StarCraft_League_Remastered/8.html":  "2019-06-30",
		"starcraft/KCM/Race_Survival/2026/1.html":                 "2026-01-15",
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
				t.Fatalf("missing expected start date for fixture %s", fx.name)
			}

			got, err := parse.StartDate(documentFromHTML(t, fx.html))
			if err != nil {
				t.Fatalf("StartDate: %v", err)
			}
			if got == nil {
				t.Fatal("StartDate returned nil")
			}
			if *got != want {
				t.Fatalf("got %q, want %q", *got, want)
			}
		})
	}
}

func TestEndDate_AllFixtures(t *testing.T) {
	t.Parallel()

	wantByFixture := map[string]string{
		"starcraft/ASL/20.html": "2025-10-26",
		"starcraft/AfreecaTV/StarCraft_League_Remastered/14.html": "2022-10-09",
		"starcraft/AfreecaTV/StarCraft_League_Remastered/8.html":  "2019-09-01",
		"starcraft/KCM/Race_Survival/2026/1.html":                 "2026-03-26",
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
				t.Fatalf("missing expected end date for fixture %s", fx.name)
			}

			got, err := parse.EndDate(documentFromHTML(t, fx.html))
			if err != nil {
				t.Fatalf("EndDate: %v", err)
			}
			if got == nil {
				t.Fatal("EndDate returned nil")
			}
			if *got != want {
				t.Fatalf("got %q, want %q", *got, want)
			}
		})
	}
}
