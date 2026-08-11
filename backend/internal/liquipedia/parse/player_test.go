package parse_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
)

func TestPlayer(t *testing.T) {
	type wantPlayer struct {
		name          string
		realName      string
		ids           []string
		preferredRace string
	}

	wantByFixture := map[string]wantPlayer{
		"starcraft/Jaedong.html": {
			name:          "Jaedong",
			realName:      "Lee Jae Dong",
			ids:           []string{"JD", "n.Die_Jaedong", "n.Die_yOngKIN"},
			preferredRace: "zerg",
		},
	}

	for _, fx := range loadFixturesForExpectations(t, wantByFixture) {
		got, err := parse.Player(fx.meta.SourceURL, fx.html)
		if err != nil {
			t.Fatalf("%s: %v", fx.name, err)
		}
		want := wantByFixture[fx.name]

		if got.Link == "" {
			t.Fatalf("%s: empty link", fx.name)
		}
		if got.Link != fx.meta.SourceURL {
			t.Errorf("%s: link=%q, want %q", fx.name, got.Link, fx.meta.SourceURL)
		}
		if got.Name == nil || *got.Name != want.name {
			t.Errorf("%s: name=%v, want %q", fx.name, got.Name, want.name)
		}
		if got.RealName == nil || *got.RealName != want.realName {
			t.Errorf("%s: realName=%v, want %q", fx.name, got.RealName, want.realName)
		}
		if got.PreferredRace == nil || *got.PreferredRace != want.preferredRace {
			t.Errorf("%s: preferredRace=%v, want %q", fx.name, got.PreferredRace, want.preferredRace)
		}
		if len(got.IDs) != len(want.ids) {
			t.Fatalf("%s: ids=%v, want %v", fx.name, got.IDs, want.ids)
		}
		for i := range want.ids {
			if got.IDs[i] != want.ids[i] {
				t.Errorf("%s: ids[%d]=%q, want %q", fx.name, i, got.IDs[i], want.ids[i])
			}
		}
	}
}
