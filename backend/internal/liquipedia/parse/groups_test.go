package parse_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

func TestGroups_FromStandingsTable(t *testing.T) {
	t.Parallel()

	html := `
<html><body>
<h3><span class="mw-headline">Round of 24</span></h3>
<h4><span class="mw-headline">Group A</span></h4>
<table class="wikitable grouptable">
  <tr>
    <td><div class="block-player"><span class="name"><a href="/starcraft/Sharp">Sharp</a></span>
      <span class="race"><img alt="Terran" /></span></div></td>
  </tr>
  <tr>
    <td><div class="block-player"><span class="name"><a href="/starcraft/Ample">Ample</a></span>
      <span class="race"><img alt="Protoss" /></span></div></td>
  </tr>
</table>
<h4><span class="mw-headline">Group B</span></h4>
<table class="wikitable grouptable">
  <tr>
    <td><div class="block-player"><span class="name"><a href="/starcraft/Flash">Flash</a></span>
      <span class="race"><img alt="Terran" /></span></div></td>
  </tr>
</table>
<h3><span class="mw-headline">Playoffs</span></h3>
<p>No standings here</p>
</body></html>`

	doc := documentFromHTML(t, html)
	got, err := parse.Groups(doc, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("groups=%d, want 2: %+v", len(got), got)
	}
	if got[0].Phase != "Round of 24" || got[0].Name != "Group A" || got[0].SortOrder != 0 {
		t.Fatalf("group0=%+v", got[0])
	}
	if len(got[0].Players) != 2 {
		t.Fatalf("group A players=%d", len(got[0].Players))
	}
	if got[0].Players[0].Name == nil || *got[0].Players[0].Name != "Sharp" {
		t.Fatalf("first player=%v", got[0].Players[0].Name)
	}
	if got[1].Name != "Group B" || got[1].SortOrder != 1 {
		t.Fatalf("group1=%+v", got[1])
	}
	if len(got[1].Players) != 1 || got[1].Players[0].Name == nil || *got[1].Players[0].Name != "Flash" {
		t.Fatalf("group B players=%+v", got[1].Players)
	}
}

func TestGroups_EmptyTablesFallBackToStage(t *testing.T) {
	t.Parallel()

	html := `<html><body><h3>Round of 24</h3><p>no tables</p></body></html>`
	results := []model.Result{
		{
			Stage:        strPtr("Round of 24 / Group A / Match 1"),
			ParticipantA: &model.Participant{Name: strPtr("Sharp"), Link: strPtr("https://liquipedia.net/starcraft/Sharp")},
			ParticipantB: &model.Participant{Name: strPtr("Ample"), Link: strPtr("https://liquipedia.net/starcraft/Ample")},
		},
		{
			Stage:        strPtr("Round of 24 / Group A / Match 2"),
			ParticipantA: &model.Participant{Name: strPtr("Sharp"), Link: strPtr("https://liquipedia.net/starcraft/Sharp")},
			ParticipantB: &model.Participant{Name: strPtr("Soulkey"), Link: strPtr("https://liquipedia.net/starcraft/Soulkey")},
		},
		{
			Stage:        strPtr("Round of 24 / Group B / Match 1"),
			ParticipantA: &model.Participant{Name: strPtr("Flash"), Link: strPtr("https://liquipedia.net/starcraft/Flash")},
			ParticipantB: &model.Participant{Name: strPtr("Jaedong"), Link: strPtr("https://liquipedia.net/starcraft/Jaedong")},
		},
		{
			Stage:        strPtr("Playoffs / Semifinals"),
			ParticipantA: &model.Participant{Name: strPtr("Sharp"), Link: strPtr("https://liquipedia.net/starcraft/Sharp")},
			ParticipantB: &model.Participant{Name: strPtr("Flash"), Link: strPtr("https://liquipedia.net/starcraft/Flash")},
		},
	}

	got, err := parse.Groups(documentFromHTML(t, html), results)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("groups=%d, want 3 (2 pools + Semifinals): %+v", len(got), got)
	}
	if got[0].Phase != "Round of 24" || got[0].Name != "Group A" {
		t.Fatalf("group0=%+v", got[0])
	}
	if len(got[0].Players) != 3 {
		t.Fatalf("group A unique players=%d, want 3", len(got[0].Players))
	}
	if got[1].Name != "Group B" || len(got[1].Players) != 2 {
		t.Fatalf("group1=%+v", got[1])
	}
	if got[2].Phase != "Playoffs" || got[2].Name != "Semifinals" {
		t.Fatalf("playoff group=%+v", got[2])
	}
}

func TestGroups_PlayoffRoundsAsGroups(t *testing.T) {
	t.Parallel()

	results := []model.Result{
		{
			Stage:        strPtr("Playoffs / Grand Final"),
			ParticipantA: &model.Participant{Name: strPtr("A"), Link: strPtr("https://liquipedia.net/starcraft/A")},
			ParticipantB: &model.Participant{Name: strPtr("B"), Link: strPtr("https://liquipedia.net/starcraft/B")},
		},
	}
	got, err := parse.Groups(documentFromHTML(t, `<html><body></body></html>`), results)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Phase != "Playoffs" || got[0].Name != "Grand Final" {
		t.Fatalf("want Playoffs/Grand Final, got %+v", got)
	}
}

func TestGroups_StandingsPreferredOverStage(t *testing.T) {
	t.Parallel()

	html := `
<html><body>
<h3>Round of 24</h3>
<h4>Group A</h4>
<table class="wikitable grouptable">
  <tr><td><div class="block-player"><span class="name"><a href="/starcraft/Only">Only</a></span></div></td></tr>
</table>
</body></html>`
	results := []model.Result{
		{
			Stage:        strPtr("Round of 24 / Group Z / Match 1"),
			ParticipantA: &model.Participant{Name: strPtr("Other")},
			ParticipantB: &model.Participant{Name: strPtr("Else")},
		},
	}
	got, err := parse.Groups(documentFromHTML(t, html), results)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Group A" {
		t.Fatalf("want standings Group A only, got %+v", got)
	}
	if got[0].Players[0].Name == nil || *got[0].Players[0].Name != "Only" {
		t.Fatalf("player=%v", got[0].Players[0].Name)
	}
}

func TestGroups_DedupCaseInsensitive(t *testing.T) {
	t.Parallel()

	html := `
<html><body>
<h3>Ro24</h3>
<h4>Group A</h4>
<table class="wikitable grouptable">
  <tr><td><div class="block-player"><span class="name"><a href="/starcraft/A">A</a></span></div></td></tr>
</table>
<h4>group a</h4>
<table class="wikitable group-table">
  <tr><td><div class="block-player"><span class="name"><a href="/starcraft/B">B</a></span></div></td></tr>
</table>
</body></html>`
	got, err := parse.Groups(documentFromHTML(t, html), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 after dedupe, got %d: %+v", len(got), got)
	}
}

func strPtr(s string) *string { return &s }

func TestGroups_AdvancingMarkersAndFallback(t *testing.T) {
	t.Parallel()

	marked := `
<html><body>
<h3>Round of 24</h3>
<h4>Group A</h4>
<table class="wikitable grouptable">
  <tr class="bg-up">
    <td><div class="block-player"><span class="name"><a href="/starcraft/Sharp">Sharp</a></span></div></td>
  </tr>
  <tr>
    <td><div class="block-player"><span class="name"><a href="/starcraft/Ample">Ample</a></span></div></td>
  </tr>
  <tr class="bg-up">
    <td><div class="block-player"><span class="name"><a href="/starcraft/Flash">Flash</a></span></div></td>
  </tr>
  <tr>
    <td><div class="block-player"><span class="name"><a href="/starcraft/Soulkey">Soulkey</a></span></div></td>
  </tr>
</table>
</body></html>`
	got, err := parse.Groups(documentFromHTML(t, marked), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Players) != 4 {
		t.Fatalf("groups=%+v", got)
	}
	want := []bool{true, false, true, false}
	for i, p := range got[0].Players {
		if p.IsWinner != want[i] {
			t.Fatalf("player %d IsWinner=%v want %v (%s)", i, p.IsWinner, want[i], nullName(p))
		}
	}

	fallback4 := `
<html><body>
<h3>Round of 24</h3>
<h4>Group A</h4>
<table class="wikitable grouptable">
  <tr><td><div class="block-player"><span class="name"><a href="/starcraft/A">A</a></span></div></td></tr>
  <tr><td><div class="block-player"><span class="name"><a href="/starcraft/B">B</a></span></div></td></tr>
  <tr><td><div class="block-player"><span class="name"><a href="/starcraft/C">C</a></span></div></td></tr>
  <tr><td><div class="block-player"><span class="name"><a href="/starcraft/D">D</a></span></div></td></tr>
</table>
</body></html>`
	got, err = parse.Groups(documentFromHTML(t, fallback4), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got[0].Players[0].IsWinner || !got[0].Players[1].IsWinner || got[0].Players[2].IsWinner || got[0].Players[3].IsWinner {
		t.Fatalf("want top 2 winners, got %+v", winnerFlags(got[0].Players))
	}

	fallback2 := `
<html><body>
<h3>Playoffs</h3>
<h4>Finals</h4>
<table class="wikitable grouptable">
  <tr><td><div class="block-player"><span class="name"><a href="/starcraft/A">A</a></span></div></td></tr>
  <tr><td><div class="block-player"><span class="name"><a href="/starcraft/B">B</a></span></div></td></tr>
</table>
</body></html>`
	got, err = parse.Groups(documentFromHTML(t, fallback2), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got[0].Players[0].IsWinner || got[0].Players[1].IsWinner {
		t.Fatalf("want top 1 winner, got %+v", winnerFlags(got[0].Players))
	}

	divMarked := `
<html><body>
<h3>Week 1</h3>
<div class="group-table">
  <div class="group-table-header"><span class="group-table-title">Standings</span></div>
  <div class="group-table-results">
    <div class="group-table-result-row bg-up">
      <div class="group-table-cell group-table-entry"><div class="block-player"><span class="name"><a href="/starcraft/Protoss">Protoss</a></span></div></div>
    </div>
    <div class="group-table-result-row">
      <div class="group-table-cell group-table-entry"><div class="block-player"><span class="name"><a href="/starcraft/Terran">Terran</a></span></div></div>
    </div>
  </div>
</div>
</body></html>`
	got, err = parse.Groups(documentFromHTML(t, divMarked), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Standings" || len(got[0].Players) != 2 {
		t.Fatalf("div group=%+v", got)
	}
	if !got[0].Players[0].IsWinner || got[0].Players[1].IsWinner {
		t.Fatalf("div winners=%+v", winnerFlags(got[0].Players))
	}

	fromResults := []model.Result{{
		Stage:        strPtr("Round of 24 / Group Z / Match 1"),
		ParticipantA: &model.Participant{Name: strPtr("X"), Link: strPtr("https://liquipedia.net/starcraft/X")},
		ParticipantB: &model.Participant{Name: strPtr("Y"), Link: strPtr("https://liquipedia.net/starcraft/Y")},
	}}
	got, err = parse.Groups(documentFromHTML(t, `<html><body></body></html>`), fromResults)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0].Players[0].IsWinner || got[0].Players[1].IsWinner {
		t.Fatalf("result-only groups should have no winners, got %+v", got)
	}
}

func nullName(p model.Participant) string {
	if p.Name == nil {
		return ""
	}
	return *p.Name
}

func winnerFlags(players []model.Participant) []bool {
	out := make([]bool, len(players))
	for i, p := range players {
		out[i] = p.IsWinner
	}
	return out
}

func TestGroups_FixtureASLIfPresent(t *testing.T) {
	t.Parallel()
	root := testdataRoot(t)
	htmlPath := filepath.Join(root, "starcraft", "ASL", "20.html")
	html, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Skip("ASL/20 fixture HTML not present")
	}
	metaPath := strings.TrimSuffix(htmlPath, ".html") + ".meta.json"
	metaRaw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Skip("ASL/20 meta not present")
	}
	var meta liquipedia.FixtureMeta
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		t.Fatal(err)
	}
	page, err := parse.Tournament(meta.SourceURL, string(html))
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Groups) < 6 {
		t.Fatalf("ASL20 groups=%d, want at least 6 (Ro24 groups)", len(page.Groups))
	}
	found := false
	for _, g := range page.Groups {
		if strings.Contains(strings.ToLower(g.Phase), "round of 24") &&
			strings.EqualFold(g.Name, "Group A") &&
			len(g.Players) >= 3 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing Round of 24 / Group A with players; groups=%+v", summarizeGroups(page.Groups))
	}
}

func summarizeGroups(groups []model.TournamentGroup) []string {
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		out = append(out, g.Phase+" / "+g.Name+" ("+strconv.Itoa(len(g.Players))+")")
	}
	return out
}
