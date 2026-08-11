package parse_test

import (
	"strings"
	"testing"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

func TestResults_AllFixtures(t *testing.T) {
	t.Parallel()

	type expect struct {
		minCount     int
		maxCount     int
		minPlayed    int
		maxPlayed    int
		first        matchSample
		lastPlayed   *matchSample
		requireFinal bool
	}

	wantByFixture := map[string]expect{
		"starcraft/ASL/20.html": {
			minCount:  55,
			maxCount:  60,
			minPlayed: 55,
			maxPlayed: 60,
			first: matchSample{
				a: "Sharp", b: "Ample", scoreA: 1, scoreB: 0, played: true,
				stageContains: "Group A",
			},
			lastPlayed: &matchSample{
				a: "SnOw", b: "soma", scoreA: 2, scoreB: 4, played: true,
				stageContains: "Finals",
			},
			requireFinal: true,
		},
		"starcraft/ASL/22.html": {
			minCount:  45,
			maxCount:  55,
			minPlayed: 0,
			maxPlayed: 0,
			first: matchSample{
				a: "Rush", b: "Hm", played: false,
				stageContains: "Group A",
			},
		},
		"starcraft/AfreecaTV/StarCraft_League_Remastered/14.html": {
			minCount:  55,
			maxCount:  65,
			minPlayed: 55,
			maxPlayed: 65,
			first: matchSample{
				played:        true,
				stageContains: "Group",
			},
			requireFinal: true,
		},
		"starcraft/AfreecaTV/StarCraft_League_Remastered/8.html": {
			minCount:  55,
			maxCount:  65,
			minPlayed: 55,
			maxPlayed: 65,
			first: matchSample{
				played:        true,
				stageContains: "Group",
			},
			requireFinal: true,
		},
		"starcraft/KCM/Race_Survival/2026/1.html": {
			minCount:  60,
			maxCount:  70,
			minPlayed: 60,
			maxPlayed: 70,
			first: matchSample{
				played:        true,
				stageContains: "Week 1",
			},
			requireFinal: true,
		},
	}

	fixtures := loadFixturesForExpectations(t, wantByFixture)

	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.name, func(t *testing.T) {
			t.Parallel()

			want, ok := wantByFixture[fx.name]
			if !ok {
				t.Fatalf("missing expected results for fixture %s", fx.name)
			}

			got, err := parse.Results(documentFromHTML(t, fx.html))
			if err != nil {
				t.Fatalf("Results: %v", err)
			}
			if len(got) < want.minCount || len(got) > want.maxCount {
				t.Fatalf("count=%d, want between %d and %d", len(got), want.minCount, want.maxCount)
			}

			played := 0
			for i, r := range got {
				if r.Order != i+1 {
					t.Fatalf("order at index %d: got %d, want %d", i, r.Order, i+1)
				}
				if r.Played {
					played++
					if r.ScoreA == nil || r.ScoreB == nil {
						t.Fatalf("order %d played but missing scores", r.Order)
					}
				}
				if r.ParticipantA == nil || r.ParticipantA.Name == nil {
					t.Fatalf("order %d missing participant A", r.Order)
				}
				if r.ParticipantB == nil || r.ParticipantB.Name == nil {
					t.Fatalf("order %d missing participant B", r.Order)
				}
			}
			if played < want.minPlayed || played > want.maxPlayed {
				t.Fatalf("played=%d, want between %d and %d", played, want.minPlayed, want.maxPlayed)
			}

			assertTimeOrder(t, got)
			assertMatchSample(t, got[0], want.first)

			if want.lastPlayed != nil {
				var last model.Result
				for i := len(got) - 1; i >= 0; i-- {
					if got[i].Played {
						last = got[i]
						break
					}
				}
				assertMatchSample(t, last, *want.lastPlayed)
			}

			if want.requireFinal {
				found := false
				for _, r := range got {
					if r.Stage != nil && (strings.Contains(strings.ToLower(*r.Stage), "finals") ||
						strings.Contains(strings.ToLower(*r.Stage), "grand final")) {
						found = true
						break
					}
				}
				if !found {
					t.Fatal("expected a Finals / Grand Final stage match")
				}
			}
		})
	}
}

type matchSample struct {
	a, b           string
	scoreA, scoreB int
	played         bool
	stageContains  string
}

func assertMatchSample(t *testing.T, r model.Result, want matchSample) {
	t.Helper()
	if r.Played != want.played {
		t.Fatalf("played=%v, want %v", r.Played, want.played)
	}
	if want.a != "" {
		if r.ParticipantA == nil || r.ParticipantA.Name == nil || *r.ParticipantA.Name != want.a {
			t.Fatalf("participantA=%v, want %q", nameOf(r.ParticipantA), want.a)
		}
	}
	if want.b != "" {
		if r.ParticipantB == nil || r.ParticipantB.Name == nil || *r.ParticipantB.Name != want.b {
			t.Fatalf("participantB=%v, want %q", nameOf(r.ParticipantB), want.b)
		}
	}
	if want.played && want.a != "" {
		if r.ScoreA == nil || *r.ScoreA != want.scoreA || r.ScoreB == nil || *r.ScoreB != want.scoreB {
			t.Fatalf("score=%v-%v, want %d-%d", r.ScoreA, r.ScoreB, want.scoreA, want.scoreB)
		}
	}
	if want.stageContains != "" {
		if r.Stage == nil || !strings.Contains(*r.Stage, want.stageContains) {
			t.Fatalf("stage=%v, want containing %q", r.Stage, want.stageContains)
		}
	}
}

func assertTimeOrder(t *testing.T, results []model.Result) {
	t.Helper()
	var prev *time.Time
	for _, r := range results {
		if r.DateTime == nil {
			continue
		}
		cur, ok := parseResultTime(*r.DateTime)
		if !ok {
			continue
		}
		if prev != nil && cur.Before(*prev) {
			t.Fatalf("order %d datetime %s is before previous timed match %s", r.Order, *r.DateTime, prev.Format(time.RFC3339))
		}
		prev = &cur
	}
}

func parseResultTime(s string) (time.Time, bool) {
	if tm, err := time.Parse(time.RFC3339, s); err == nil {
		return tm, true
	}
	if tm, err := time.Parse("2006-01-02", s); err == nil {
		return tm, true
	}
	return time.Time{}, false
}

func nameOf(p *model.Participant) string {
	if p == nil || p.Name == nil {
		return "<nil>"
	}
	return *p.Name
}
