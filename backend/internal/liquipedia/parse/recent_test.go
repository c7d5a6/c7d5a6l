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

func TestRecentTournaments_table2(t *testing.T) {
	html := `
		<div class="table2 table2--generic tournaments-listing">
			<table class="table2__table">
				<tr class="table2__row--head">
					<th>Tier</th>
					<th colspan="2">Tournament</th>
					<th>Date</th>
					<th>Prize Pool</th>
					<th class="column__placement">Winner</th>
				</tr>
				<tr class="table2__row--body">
					<td><a href="/starcraft/Minor_Tournaments">Showm. (Min.)</a></td>
					<td><span class="league-icon-small-image"><a href="/starcraft/K-League"><img alt=""></a></span></td>
					<td class="column__tournament">
						<a href="/starcraft/Daily_Proleague/K-League/2026-08-30">K-League: August 30, 2026</a>
					</td>
					<td>Aug 30, 2026</td>
					<td>$5,046.03</td>
					<td class="column__placement"><a href="/starcraft/Jaedong">Jaedong</a></td>
				</tr>
				<tr class="table2__row--body">
					<td><a href="/starcraft/Premier_Tournaments">Premier</a></td>
					<td></td>
					<td class="column&#95;&#95;tournament">
						<a href="/starcraft/ASL/22">ASL Season 22</a>
					</td>
					<td>Aug 29–30, 2026</td>
					<td>$10,000</td>
					<td class="column&#95;&#95;placement"><a href="/starcraft/Flash">Flash</a></td>
				</tr>
				<tr class="table2__row--body">
					<td>Major</td>
					<td></td>
					<td class="column__tournament"><a href="/starcraft/KCM/Race_Survival/2026/1">KCM Race Survival 2026 Season 1</a></td>
					<td>May 30 – Aug 29, 2026</td>
					<td>-</td>
					<td class="column__placement"><a href="/starcraft/Soulkey">Soulkey</a></td>
				</tr>
			</table>
		</div>
	`

	got, err := parse.RecentTournaments(html)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d want 3: %+v", len(got), got)
	}
	if got[0].Link != "https://liquipedia.net/starcraft/Daily_Proleague/K-League/2026-08-30" {
		t.Errorf("link=%q", got[0].Link)
	}
	if got[0].StartDate == nil || *got[0].StartDate != "2026-08-30" {
		t.Errorf("start=%v", got[0].StartDate)
	}
	if got[1].StartDate == nil || *got[1].StartDate != "2026-08-29" {
		t.Errorf("range start=%v", got[1].StartDate)
	}
	if got[1].EndDate == nil || *got[1].EndDate != "2026-08-30" {
		t.Errorf("range end=%v", got[1].EndDate)
	}
	if got[2].StartDate == nil || *got[2].StartDate != "2026-05-30" {
		t.Errorf("cross start=%v", got[2].StartDate)
	}
	if got[2].EndDate == nil || *got[2].EndDate != "2026-08-29" {
		t.Errorf("cross end=%v", got[2].EndDate)
	}
	if got[0].Tier == nil || *got[0].Tier != "Showm. (Min.)" {
		t.Errorf("tier=%v", got[0].Tier)
	}
	for _, row := range got {
		switch row.Link {
		case "https://liquipedia.net/starcraft/Jaedong",
			"https://liquipedia.net/starcraft/Flash",
			"https://liquipedia.net/starcraft/Soulkey",
			"https://liquipedia.net/starcraft/Minor_Tournaments",
			"https://liquipedia.net/starcraft/Premier_Tournaments":
			t.Fatalf("picked non-event link %s", row.Link)
		}
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
	if len(got) < 50 {
		t.Fatalf("fixture listings=%d, want at least 50", len(got))
	}

	foundKLeague := false
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
		if strings.Contains(row.Link, "/Jaedong") && row.Name == "Jaedong" {
			t.Fatalf("picked winner profile as listing: %s", row.Link)
		}
		if strings.Contains(strings.ToLower(ref.Title), "daily_proleague/k-league/2026-08-30") {
			foundKLeague = true
			if row.StartDate == nil || *row.StartDate != "2026-08-30" {
				t.Errorf("k-league start=%v", row.StartDate)
			}
		}
	}
	if !foundKLeague {
		t.Fatal("fixture missing Daily Proleague/K-League/2026-08-30")
	}
}
