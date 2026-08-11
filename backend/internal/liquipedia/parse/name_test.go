package parse_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
)

func TestName_AllFixtures(t *testing.T) {
	t.Parallel()

	wantByFixture := map[string]string{
		"starcraft/ASL/20.html": "ASL Season 20",
		"starcraft/ASL/22.html": "ASL Season 22",
		"starcraft/AfreecaTV/StarCraft_League_Remastered/14.html": "ASL Season 14",
		"starcraft/AfreecaTV/StarCraft_League_Remastered/8.html":  "ASL Season 8",
		"starcraft/KCM/Race_Survival/2026/1.html":                 "KCM Race Survival 2026 Season 1",
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
				t.Fatalf("missing expected name for fixture %s", fx.name)
			}

			got, err := parse.Name(documentFromHTML(t, fx.html))
			if err != nil {
				t.Fatalf("Name: %v", err)
			}
			if got == nil {
				t.Fatal("Name returned nil")
			}
			if *got != want {
				t.Fatalf("got %q, want %q", *got, want)
			}
		})
	}
}
