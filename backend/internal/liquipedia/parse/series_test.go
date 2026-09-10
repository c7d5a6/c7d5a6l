package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestSeriesPlayed(t *testing.T) {
	t.Parallel()
	one, zero, two := 1, 0, 2
	cases := []struct {
		a, b   *int
		bestOf int
		want   bool
	}{
		{nil, nil, 3, false},
		{&one, &zero, 1, true},
		{&one, &zero, 0, false},
		{&two, &zero, 0, true},
		{&one, &zero, 3, false},
		{&two, &zero, 3, true},
		{&two, &one, 3, true},
		{&one, &one, 3, false},
		{&two, &one, 5, false},
		{&two, &two, 5, false},
		{ptr(3), &one, 5, true},
		{ptr(4), &two, 7, true},
		{ptr(3), &two, 7, false},
	}
	for _, tc := range cases {
		got := seriesPlayed(tc.a, tc.b, tc.bestOf)
		if got != tc.want {
			t.Fatalf("seriesPlayed(%v,%v,bo%d)=%v want %v", tc.a, tc.b, tc.bestOf, got, tc.want)
		}
	}
}

func ptr(n int) *int { return &n }

func TestParseBestOfN(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want int
	}{
		{"(Bo3)", 3},
		{"Bo1", 1},
		{"Best of 7", 7},
		{"Round of 16 (Bo3)", 3},
		{"Round of 16", 0},
		{"", 0},
	}
	for _, tc := range cases {
		if got := parseBestOfN(tc.in); got != tc.want {
			t.Fatalf("parseBestOfN(%q)=%d want %d", tc.in, got, tc.want)
		}
	}
}

func TestMatchBestOfFromScoreholder(t *testing.T) {
	t.Parallel()
	html := `
<div class="brkts-matchlist-match">
  <div class="brkts-matchlist-score"><div class="brkts-matchlist-cell-content">1</div></div>
  <div class="brkts-matchlist-score"><div class="brkts-matchlist-cell-content">0</div></div>
  <div class="brkts-match-info-popup">
    <span class="match-info-header-scoreholder-lower">(Bo3)</span>
  </div>
</div>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	match := doc.Find(".brkts-matchlist-match")
	popup := match.Find(".brkts-match-info-popup")
	n := matchBestOf(match, popup)
	if n != 3 {
		t.Fatalf("bestOf=%d want 3", n)
	}
	one, zero := 1, 0
	if seriesPlayed(&one, &zero, n) {
		t.Fatal("Bo3 1:0 must not be played")
	}
}

func TestMatchlistParse_bo3OneZeroNotPlayed(t *testing.T) {
	t.Parallel()
	html := `
<html><body>
<div class="brkts-matchlist">
<div class="brkts-matchlist-match">
  <div class="brkts-matchlist-opponent"><span class="name">Shuttle</span></div>
  <div class="brkts-matchlist-score"><div class="brkts-matchlist-cell-content">1</div></div>
  <div class="brkts-matchlist-score"><div class="brkts-matchlist-cell-content">0</div></div>
  <div class="brkts-matchlist-opponent"><span class="name">Rush</span></div>
  <div class="brkts-match-info-popup">
    <span class="timer-object" data-timestamp="1000"></span>
    <span class="match-info-header-scoreholder-lower">(Bo3)</span>
    <div class="match-info-header-opponent"><span class="name"><a href="/starcraft/Shuttle">Shuttle</a></span></div>
    <div class="match-info-header-opponent"><span class="name"><a href="/starcraft/Rush">Rush</a></span></div>
  </div>
</div>
</div>
</body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Results(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("results=%d", len(got))
	}
	if got[0].Played {
		t.Fatalf("Bo3 1:0 played=%v scores=%v:%v", got[0].Played, got[0].ScoreA, got[0].ScoreB)
	}
}

func TestMatchlistParse_bo3TwoOnePlayed(t *testing.T) {
	t.Parallel()
	html := `
<html><body>
<div class="brkts-matchlist">
<div class="brkts-matchlist-match">
  <div class="brkts-matchlist-opponent"><span class="name">Shuttle</span></div>
  <div class="brkts-matchlist-score"><div class="brkts-matchlist-cell-content">2</div></div>
  <div class="brkts-matchlist-score"><div class="brkts-matchlist-cell-content">1</div></div>
  <div class="brkts-matchlist-opponent"><span class="name">Rush</span></div>
  <div class="brkts-match-info-popup">
    <span class="timer-object" data-timestamp="1000" data-finished="finished"></span>
    <span class="match-info-header-scoreholder-lower">(Bo3)</span>
    <div class="match-info-header-opponent"><span class="name"><a href="/starcraft/Shuttle">Shuttle</a></span></div>
    <div class="match-info-header-opponent"><span class="name"><a href="/starcraft/Rush">Rush</a></span></div>
  </div>
</div>
</div>
</body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Results(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Played {
		t.Fatalf("Bo3 2:1 want played, got %+v", got)
	}
}

func TestMatchlistParse_finishedBo1WithoutTokenPlayed(t *testing.T) {
	t.Parallel()
	html := `
<html><body>
<div class="brkts-matchlist">
<div class="brkts-matchlist-match">
  <div class="brkts-matchlist-opponent"><span class="name">Sharp</span></div>
  <div class="brkts-matchlist-score"><div class="brkts-matchlist-cell-content">1</div></div>
  <div class="brkts-matchlist-score"><div class="brkts-matchlist-cell-content">0</div></div>
  <div class="brkts-matchlist-opponent"><span class="name">Ample</span></div>
  <div class="brkts-match-info-popup">
    <span class="timer-object" data-timestamp="1000" data-finished="finished"></span>
    <div class="match-info-header-opponent"><span class="name"><a href="/starcraft/Sharp">Sharp</a></span></div>
    <div class="match-info-header-opponent"><span class="name"><a href="/starcraft/Ample">Ample</a></span></div>
  </div>
</div>
</div>
</body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Results(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Played {
		t.Fatalf("finished 1:0 without Bo token want played, got %+v", got)
	}
}

func TestMatchlistParse_liveOneZeroUnknownNotPlayed(t *testing.T) {
	t.Parallel()
	html := `
<html><body>
<div class="brkts-matchlist">
<div class="brkts-matchlist-match">
  <div class="brkts-matchlist-opponent"><span class="name">Shuttle</span></div>
  <div class="brkts-matchlist-score"><div class="brkts-matchlist-cell-content">1</div></div>
  <div class="brkts-matchlist-score"><div class="brkts-matchlist-cell-content">0</div></div>
  <div class="brkts-matchlist-opponent"><span class="name">Rush</span></div>
  <div class="brkts-match-info-popup">
    <span class="timer-object" data-timestamp="1000"></span>
    <div class="match-info-header-opponent"><span class="name"><a href="/starcraft/Shuttle">Shuttle</a></span></div>
    <div class="match-info-header-opponent"><span class="name"><a href="/starcraft/Rush">Rush</a></span></div>
  </div>
</div>
</div>
</body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Results(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("results=%d", len(got))
	}
	if got[0].Played {
		t.Fatal("live 1:0 with unknown format must not be played")
	}
}

func TestBracketParse_winnerClassOneZeroPlayed(t *testing.T) {
	t.Parallel()
	html := `
<html><body>
<div class="brkts-bracket">
<div class="brkts-match brkts-match-popup-wrapper">
  <div class="brkts-opponent-entry brkts-opponent-hover" aria-label="Ggaemo">
    <div class="brkts-opponent-entry-left brkts-opponent-win Zerg">
      <span class="name">ggaemo</span>
    </div>
    <div class="brkts-opponent-score-outer"><div class="brkts-opponent-score-inner"><b>1</b></div></div>
  </div>
  <div class="brkts-opponent-entry brkts-opponent-entry-last brkts-opponent-hover" aria-label="Shine">
    <div class="brkts-opponent-entry-left Zerg">
      <span class="name">Shine</span>
    </div>
    <div class="brkts-opponent-score-outer"><div class="brkts-opponent-score-inner">0</div></div>
  </div>
</div>
</div>
</body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Results(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("results=%d", len(got))
	}
	r := got[0]
	if !r.Played {
		t.Fatal("bracket 1:0 with winner class must be played")
	}
	if r.ScoreA == nil || *r.ScoreA != 1 || r.ScoreB == nil || *r.ScoreB != 0 {
		t.Fatalf("score=%v:%v want 1:0", r.ScoreA, r.ScoreB)
	}
	if r.ParticipantA == nil || r.ParticipantA.Name == nil || *r.ParticipantA.Name != "ggaemo" {
		t.Fatalf("a=%v", r.ParticipantA)
	}
	if r.ParticipantB == nil || r.ParticipantB.Name == nil || *r.ParticipantB.Name != "Shine" {
		t.Fatalf("b=%v", r.ParticipantB)
	}
}

func TestBracketParse_bo3OneOneNotPlayedKeepsScore(t *testing.T) {
	t.Parallel()
	html := `
<html><body>
<div class="brkts-bracket">
<div class="brkts-match brkts-match-popup-wrapper">
  <div class="brkts-opponent-entry" aria-label="Ggaemo">
    <div class="brkts-opponent-entry-left Zerg"><span class="name">ggaemo</span></div>
    <div class="brkts-opponent-score-outer"><div class="brkts-opponent-score-inner">1</div></div>
  </div>
  <div class="brkts-opponent-entry" aria-label="Shine">
    <div class="brkts-opponent-entry-left Zerg"><span class="name">Shine</span></div>
    <div class="brkts-opponent-score-outer"><div class="brkts-opponent-score-inner">1</div></div>
  </div>
  <div class="brkts-match-info-popup">
    <span class="match-info-header-scoreholder-lower">(Bo3)</span>
  </div>
</div>
</div>
</body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Results(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("results=%d", len(got))
	}
	r := got[0]
	if r.Played {
		t.Fatal("Bo3 1:1 must not be played")
	}
	if r.ScoreA == nil || *r.ScoreA != 1 || r.ScoreB == nil || *r.ScoreB != 1 {
		t.Fatalf("score=%v:%v want 1:1", r.ScoreA, r.ScoreB)
	}
}
