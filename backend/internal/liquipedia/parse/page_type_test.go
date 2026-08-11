package parse_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
)

func TestDetectPageType(t *testing.T) {
	wantByFixture := map[string]parse.PageType{
		"starcraft/ASL/20.html":                                     parse.PageTypeTournament,
		"starcraft/ASL/22.html":                                     parse.PageTypeTournament,
		"starcraft/AfreecaTV/StarCraft_League_Remastered/14.html":   parse.PageTypeTournament,
		"starcraft/AfreecaTV/StarCraft_League_Remastered/8.html":    parse.PageTypeTournament,
		"starcraft/KCM/Race_Survival/2026/1.html":                   parse.PageTypeTournament,
		"starcraft/Jaedong.html":                                    parse.PageTypePlayer,
		"starcraft/Evil_Geniuses.html":                              parse.PageTypeUnknown,
	}

	for _, fx := range loadFixturesForExpectations(t, wantByFixture) {
		got, err := parse.PageTypeFromHTML(fx.html)
		if err != nil {
			t.Errorf("%s: %v", fx.name, err)
			continue
		}
		if got != wantByFixture[fx.name] {
			t.Errorf("%s: page type=%q, want %q", fx.name, got, wantByFixture[fx.name])
		}
	}
}

func TestDetectPageType_fallbackTemplate(t *testing.T) {
	html := `<div class="fo-nttax-infobox"><a href="/starcraft/Template:Infobox_player" title="Template:Infobox player">h</a></div>`
	got, err := parse.PageTypeFromHTML(html)
	if err != nil {
		t.Fatal(err)
	}
	if got != parse.PageTypePlayer {
		t.Fatalf("got %q, want player", got)
	}
}

func TestDetectPageType_empty(t *testing.T) {
	got, err := parse.PageTypeFromHTML(`<p>no infobox</p>`)
	if err != nil {
		t.Fatal(err)
	}
	if got != parse.PageTypeUnknown {
		t.Fatalf("got %q, want unknown", got)
	}
}
