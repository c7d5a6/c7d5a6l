package parse_test

import (
	"strings"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
)

func TestParticipants_LastHerOLocalRedlink(t *testing.T) {
	t.Parallel()
	fx := loadFixturesNamed(t, map[string]struct{}{
		"starcraft/ASL/22/Qualifier/Day_1.html": {},
	})[0]
	got, err := parse.Participants(documentFromHTML(t, fx.html))
	if err != nil {
		t.Fatal(err)
	}
	var p *struct {
		name, realName, link string
	}
	for _, part := range got {
		if part.Name != nil && *part.Name == "LastHerO" {
			p = &struct {
				name, realName, link string
			}{name: *part.Name}
			if part.RealName != nil {
				p.realName = *part.RealName
			}
			if part.Link != nil {
				p.link = *part.Link
			}
			break
		}
	}
	if p == nil {
		t.Fatal("missing LastHerO participant")
	}
	if p.realName != "김홍철" {
		t.Fatalf("realName=%q, want 김홍철", p.realName)
	}
	wantLink := liquipedia.LocalPlayerURL("starcraft", "LastHerO")
	if p.link != wantLink {
		t.Fatalf("link=%q, want %q", p.link, wantLink)
	}
}

func TestParticipants_LinkedPlayerKeepsCombinedLabel(t *testing.T) {
	t.Parallel()
	html := `<html><body>
<div class="participantTable participantTable-faction">
  <div class="participantTable-row">
    <div class="participantTable-faction-header participantTable-entry Terran"><div>Terran</div></div>
  </div>
  <div class="participantTable-row">
    <div class="participantTable-entry">
      <div class="block-player">
        <span class="name"><a href="/starcraft/Bisu">Bisu 이준석</a></span>
      </div>
    </div>
  </div>
</div>
</body></html>`
	got, err := parse.Participants(documentFromHTML(t, html))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("participants=%d", len(got))
	}
	if got[0].Name == nil || *got[0].Name != "Bisu 이준석" {
		t.Fatalf("name=%v", got[0].Name)
	}
	if got[0].RealName != nil {
		t.Fatalf("realName=%v, want nil for linked player", got[0].RealName)
	}
	if got[0].Link == nil || !strings.HasPrefix(*got[0].Link, "https://liquipedia.net/") {
		t.Fatalf("link=%v", got[0].Link)
	}
}
