package parse_test

import (
	"strings"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
)

func TestRecentTournaments_wikiTable(t *testing.T) {
	html := `
		<h2><span class="mw-headline">Ongoing</span></h2>
		<table class="wikitable">
			<tr><th>Date</th><th>Tier</th><th>Tournament</th></tr>
			<tr>
				<td>2026-04-01 - 2026-04-15</td>
				<td><a href="/starcraft/Premier_Tournaments">Premier</a></td>
				<td>
					<span class="league-icon-small-image"><a href="/starcraft/ASL"><img alt=""></a></span>
					<a href="/starcraft/ASL/22">ASL Season 22</a>
				</td>
			</tr>
			<tr>
				<td>2026-05-01</td>
				<td>Major</td>
				<td><a href="/starcraft/KCM/Race_Survival/2026/1">KCM Race Survival 2026 Season 1</a></td>
			</tr>
		</table>
		<h2><span class="mw-headline">Completed</span></h2>
		<div class="gridTable">
			<div class="gridRow gridHeader">
				<div class="gridCell Date">Date</div>
				<div class="gridCell Tournament">Tournament</div>
			</div>
			<div class="gridRow">
				<div class="gridCell Date">2026-03-01 - 2026-03-20</div>
				<div class="gridCell Tournament"><a href="/starcraft/ASL/20">ASL Season 20</a></div>
			</div>
		</div>
	`

	got, err := parse.RecentTournaments(html)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d want 3: %+v", len(got), got)
	}

	wantLinks := []string{
		"https://liquipedia.net/starcraft/ASL/22",
		"https://liquipedia.net/starcraft/KCM/Race_Survival/2026/1",
		"https://liquipedia.net/starcraft/ASL/20",
	}
	for i, want := range wantLinks {
		if got[i].Link != want {
			t.Errorf("item %d link=%q want %q", i, got[i].Link, want)
		}
	}
	if got[0].Name != "ASL Season 22" {
		t.Errorf("name=%q", got[0].Name)
	}
	if got[0].StartDate == nil || *got[0].StartDate != "2026-04-01" {
		t.Errorf("start=%v", got[0].StartDate)
	}
	if got[0].EndDate == nil || *got[0].EndDate != "2026-04-15" {
		t.Errorf("end=%v", got[0].EndDate)
	}
	if got[0].Tier == nil || *got[0].Tier != "Premier" {
		t.Errorf("tier=%v", got[0].Tier)
	}
	if got[0].Section == nil || *got[0].Section != "Ongoing" {
		t.Errorf("section=%v", got[0].Section)
	}
	if got[2].Section == nil || *got[2].Section != "Completed" {
		t.Errorf("grid section=%v", got[2].Section)
	}
}

func TestRecentTournaments_fixture(t *testing.T) {
	fixtures := loadFixturesNamed(t, map[string]struct{}{
		"starcraft/Leagues/Recent_Tournaments.html": {},
	})
	got, err := parse.RecentTournaments(fixtures[0].html)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 8 {
		t.Fatalf("fixture listings=%d, want at least 8", len(got))
	}

	seen := map[string]struct{}{}
	for _, row := range got {
		if row.Link == "" || row.Name == "" {
			t.Fatalf("empty listing %+v", row)
		}
		if _, ok := seen[row.Link]; ok {
			t.Fatalf("duplicate link %s", row.Link)
		}
		seen[row.Link] = struct{}{}
		ref, err := liquipedia.ParsePageRef(row.Link)
		if err != nil {
			t.Fatalf("link %s: %v", row.Link, err)
		}
		if ref.Wiki != "starcraft" {
			t.Fatalf("wiki=%s for %s", ref.Wiki, row.Link)
		}
		if strings.EqualFold(ref.Title, "Leagues/Recent_Tournaments") {
			t.Fatalf("listing included itself: %s", row.Link)
		}
	}
}
