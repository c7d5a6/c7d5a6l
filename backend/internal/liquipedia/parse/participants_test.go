package parse_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

func TestParticipants_AllFixtures(t *testing.T) {
	t.Parallel()

	type expect struct {
		minCount       int
		maxCount       int
		excludedNames  []string
		sample         map[string]sampleParticipant
	}

	wantByFixture := map[string]expect{
		"starcraft/ASL/20.html": {
			minCount: 28,
			maxCount: 32,
			sample: map[string]sampleParticipant{
				"Soulkey": {race: "zerg", linkSuffix: "/starcraft/Soulkey", excluded: false},
			},
		},
		"starcraft/ASL/22.html": {
			minCount:      28,
			maxCount:      30,
			excludedNames: []string{"Flash"},
			sample: map[string]sampleParticipant{
				"Flash": {race: "terran", linkSuffix: "/starcraft/Flash", excluded: true},
				"SnOw":  {race: "protoss", linkSuffix: "/starcraft/SnOw", excluded: false},
				"Bisu":  {race: "protoss", linkSuffix: "/starcraft/Bisu", excluded: false},
			},
		},
		"starcraft/AfreecaTV/StarCraft_League_Remastered/14.html": {
			minCount:      28,
			maxCount:      30,
			excludedNames: []string{"Rain"},
			sample: map[string]sampleParticipant{
				"Rain": {race: "protoss", linkSuffix: "/starcraft/Rain", excluded: true},
				"Bisu": {race: "protoss", linkSuffix: "/starcraft/Bisu", excluded: false},
			},
		},
		"starcraft/AfreecaTV/StarCraft_League_Remastered/8.html": {
			minCount: 28,
			maxCount: 29, // includes EffOrt (footnote forfeit, no strikethrough markup)
			sample: map[string]sampleParticipant{
				"Mini":  {race: "protoss", linkSuffix: "/starcraft/Mini", excluded: false},
				"FlaSh": {race: "terran", linkSuffix: "/starcraft/Flash", excluded: false},
			},
		},
		"starcraft/KCM/Race_Survival/2026/1.html": {
			minCount: 31,
			maxCount: 31,
			excludedNames: []string{"huro", "Flash"},
			sample: map[string]sampleParticipant{
				"Bisu":  {race: "protoss", linkSuffix: "/starcraft/Bisu", excluded: false},
				"huro":  {race: "protoss", linkSuffix: "/starcraft/Huro", excluded: true},
				"Flash": {race: "terran", linkSuffix: "/starcraft/Flash", excluded: true},
				"soma":  {race: "zerg", linkSuffix: "/starcraft/Soma", excluded: false},
			},
		},
	}

	fixtures := loadFixturesForExpectations(t, wantByFixture)

	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.name, func(t *testing.T) {
			t.Parallel()

			want, ok := wantByFixture[fx.name]
			if !ok {
				t.Fatalf("missing expected participants for fixture %s", fx.name)
			}

			got, err := parse.Participants(documentFromHTML(t, fx.html))
			if err != nil {
				t.Fatalf("Participants: %v", err)
			}
			if len(got) < want.minCount || len(got) > want.maxCount {
				t.Fatalf("count=%d, want between %d and %d", len(got), want.minCount, want.maxCount)
			}

			byName := map[string]model.Participant{}
			for _, p := range got {
				if p.Name == nil || *p.Name == "" {
					t.Fatal("participant missing name")
				}
				if p.Race == nil || *p.Race == "" {
					t.Fatalf("%q missing race", *p.Name)
				}
				byName[*p.Name] = p
			}

			for _, name := range want.excludedNames {
				p, ok := byName[name]
				if !ok {
					t.Fatalf("expected excluded participant %q", name)
				}
				if !p.Excluded {
					t.Fatalf("%q: excluded=false, want true", name)
				}
			}

			for name, sample := range want.sample {
				p, ok := byName[name]
				if !ok {
					t.Fatalf("missing sample participant %q (have %d)", name, len(byName))
				}
				if p.Race == nil || *p.Race != sample.race {
					t.Fatalf("%q race=%v, want %q", name, p.Race, sample.race)
				}
				if p.Excluded != sample.excluded {
					t.Fatalf("%q excluded=%v, want %v", name, p.Excluded, sample.excluded)
				}
				if sample.linkSuffix != "" {
					if p.Link == nil || !stringsHasSuffix(*p.Link, sample.linkSuffix) {
						t.Fatalf("%q link=%v, want suffix %q", name, p.Link, sample.linkSuffix)
					}
					if sample.localLink {
						if !stringsHasPrefix(*p.Link, "local://") {
							t.Fatalf("%q link should be local://, got %q", name, *p.Link)
						}
					} else if !stringsHasPrefix(*p.Link, "https://liquipedia.net/") {
						t.Fatalf("%q link should be absolute liquipedia URL, got %q", name, *p.Link)
					}
				}
			}
		})
	}
}

func TestParticipants_VANTMissingPages(t *testing.T) {
	t.Parallel()
	fx := loadFixturesNamed(t, map[string]struct{}{
		"starcraft/VANT36.5_National_Starleague.html": {},
	})[0]
	got, err := parse.Participants(documentFromHTML(t, fx.html))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 30 {
		t.Fatalf("count=%d, want >= 30", len(got))
	}
	byName := map[string]model.Participant{}
	for _, p := range got {
		if p.Name != nil {
			byName[*p.Name] = p
		}
	}
	for _, name := range []string{"Jeco", "Ever)P(NaBi"} {
		p, ok := byName[name]
		if !ok {
			t.Fatalf("missing redlink participant %q", name)
		}
		if p.Link == nil || !stringsHasPrefix(*p.Link, "local://starcraft/player/") {
			t.Fatalf("%q link=%v, want local://starcraft/player/…", name, p.Link)
		}
	}
}

type sampleParticipant struct {
	race       string
	linkSuffix string
	excluded   bool
	localLink  bool
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
