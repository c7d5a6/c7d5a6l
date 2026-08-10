package parse_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
)

func TestLiquipediaTier_AllFixtures(t *testing.T) {
	t.Parallel()

	wantByFixture := map[string]string{
		"starcraft/ASL/20.html": "Premier",
		"starcraft/AfreecaTV/StarCraft_League_Remastered/14.html": "Premier",
		"starcraft/AfreecaTV/StarCraft_League_Remastered/8.html":  "Premier",
		"starcraft/KCM/Race_Survival/2026/1.html":                 "Major",
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
				t.Fatalf("missing expected liquipedia tier for fixture %s", fx.name)
			}

			got, err := parse.LiquipediaTier(documentFromHTML(t, fx.html))
			if err != nil {
				t.Fatalf("LiquipediaTier: %v", err)
			}
			if got == nil {
				t.Fatal("LiquipediaTier returned nil")
			}
			if *got != want {
				t.Fatalf("got %q, want %q", *got, want)
			}
		})
	}
}
