package parse_test

import (
	"strings"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
)

func TestNewFixtures_TeamAndQualifierPages(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		minParticipants int
		maxParticipants int
		minResults      int
		maxResults      int
		minGroups       int
		stageContains   string
		samplePlayers   []string
	}{
		"starcraft/Daily_Proleague/K-League/2026-09-03.html": {
			minParticipants: 10,
			maxParticipants: 10,
			minResults:      15,
			maxResults:      15,
			stageContains:   "Team A vs Team B",
			samplePlayers:   []string{"Rush", "RoyaL", "Shuttle"},
		},
		"starcraft/Daily_Proleague/Hybrid_Proleague/2026-08-29.html": {
			minParticipants: 10,
			maxParticipants: 10,
			minResults:      16,
			maxResults:      16,
			stageContains:   "Team A vs Team B",
		},
		"starcraft/KCM/Special/1.html": {
			minParticipants: 10,
			maxParticipants: 10,
			minResults:      8,
			maxResults:      8,
			stageContains:   "China vs South Korea",
			samplePlayers:   []string{"Mihu", "Horang2", "Cococn"},
		},
		"starcraft/ASL/22/Qualifier/Day_1.html": {
			minParticipants: 115,
			maxParticipants: 125,
			minResults:      100,
			maxResults:      120,
			minGroups:       12,
			stageContains:   "Group 1",
			samplePlayers:   []string{"Rush", "Soulkey"},
		},
	}

	names := make(map[string]struct{}, len(cases))
	for k := range cases {
		names[k] = struct{}{}
	}
	for _, fx := range loadFixturesNamed(t, names) {
		fx := fx
		want := cases[fx.name]
		t.Run(fx.name, func(t *testing.T) {
			t.Parallel()
			doc := documentFromHTML(t, fx.html)

			participants, err := parse.Participants(doc)
			if err != nil {
				t.Fatal(err)
			}
			if len(participants) < want.minParticipants || len(participants) > want.maxParticipants {
				t.Fatalf("participants=%d, want %d-%d", len(participants), want.minParticipants, want.maxParticipants)
			}

			results, err := parse.Results(doc)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) < want.minResults || len(results) > want.maxResults {
				t.Fatalf("results=%d, want %d-%d", len(results), want.minResults, want.maxResults)
			}
			if want.stageContains != "" {
				found := false
				for _, r := range results {
					if r.Stage != nil && strings.Contains(*r.Stage, want.stageContains) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("no result stage containing %q", want.stageContains)
				}
			}
			for _, r := range results {
				if !r.Played {
					continue
				}
				if r.ParticipantA == nil || r.ParticipantA.Name == nil || r.ParticipantB == nil || r.ParticipantB.Name == nil {
					t.Fatalf("played result missing participants: %+v", r)
				}
				if r.ScoreA == nil || r.ScoreB == nil {
					t.Fatalf("played result missing scores: %+v", r)
				}
			}

			groups, err := parse.Groups(doc, results)
			if err != nil {
				t.Fatal(err)
			}
			if len(groups) < want.minGroups {
				t.Fatalf("groups=%d, want >= %d", len(groups), want.minGroups)
			}

			if len(want.samplePlayers) > 0 {
				byName := map[string]bool{}
				for _, p := range participants {
					if p.Name != nil {
						byName[*p.Name] = true
					}
				}
				for _, name := range want.samplePlayers {
					if !byName[name] {
						t.Fatalf("missing sample participant %q", name)
					}
				}
			}
		})
	}
}
