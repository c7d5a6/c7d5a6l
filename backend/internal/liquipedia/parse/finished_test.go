package parse_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
)

func TestFinished_AllFixtures(t *testing.T) {
	t.Parallel()

	wantByFixture := map[string]bool{
		"starcraft/ASL/20.html": true,
		"starcraft/ASL/22.html": false,
		"starcraft/AfreecaTV/StarCraft_League_Remastered/14.html": true,
		"starcraft/AfreecaTV/StarCraft_League_Remastered/8.html":  true,
		"starcraft/KCM/Race_Survival/2026/1.html":                 false,
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
				t.Fatalf("missing expected finished flag for fixture %s", fx.name)
			}

			got, err := parse.Finished(documentFromHTML(t, fx.html))
			if err != nil {
				t.Fatalf("Finished: %v", err)
			}
			if got == nil {
				t.Fatal("Finished returned nil")
			}
			if *got != want {
				t.Fatalf("got %v, want %v", *got, want)
			}
		})
	}
}
