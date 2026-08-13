package parse

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// Player parses a Liquipedia player HTML page into the domain model.
func Player(link string, html string) (model.PlayerPage, error) {
	debuglog.Printf("parse.Player link=%s htmlBytes=%d", link, len(html))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return model.PlayerPage{}, fmt.Errorf("parse html: %w", err)
	}

	page := model.NewPlayerPage(link)

	name, err := playerNickName(doc)
	if err != nil {
		return model.PlayerPage{}, fmt.Errorf("parse name: %w", err)
	}
	page.Name = name

	realName, err := playerRealName(doc)
	if err != nil {
		return model.PlayerPage{}, fmt.Errorf("parse real name: %w", err)
	}
	page.RealName = realName

	ids, err := playerIDs(doc)
	if err != nil {
		return model.PlayerPage{}, fmt.Errorf("parse ids: %w", err)
	}
	page.IDs = ids

	race, err := playerPreferredRace(doc)
	if err != nil {
		return model.PlayerPage{}, fmt.Errorf("parse preferred race: %w", err)
	}
	page.PreferredRace = race

	page.PortraitURL = playerPortraitURL(doc)
	debuglog.Printf("parse.Player name=%s realName=%s race=%s portrait=%s ids=%v",
		debuglog.Str(name), debuglog.Str(realName), debuglog.Str(race), debuglog.Str(page.PortraitURL), ids)

	return page, nil
}

// playerNickName is the ID shown in the primary infobox header (not the civil name).
func playerNickName(doc *goquery.Document) (*string, error) {
	return Name(doc)
}

// playerRealName prefers Romanized Name, then native Name.
func playerRealName(doc *goquery.Document) (*string, error) {
	if v, err := infoboxValue(doc, "Romanized Name"); err != nil {
		return nil, err
	} else if v != nil {
		return v, nil
	}
	return infoboxValue(doc, "Name")
}

func playerIDs(doc *goquery.Document) ([]string, error) {
	raw, err := infoboxValue(doc, "Alternate IDs")
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return []string{}, nil
	}
	return splitCommaList(*raw), nil
}

func playerPreferredRace(doc *goquery.Document) (*string, error) {
	raw, err := infoboxValue(doc, "Race")
	if err != nil {
		return nil, err
	}
	if raw == nil {
		// Fallback: race icon in the header.
		alt := doc.Find(".fo-nttax-infobox .infobox-header img").First().AttrOr("alt", "")
		if r := normalizeRace(alt); r != "" {
			return &r, nil
		}
		return nil, nil
	}
	if r := normalizeRace(*raw); r != "" {
		return &r, nil
	}
	return nil, nil
}

// playerPortraitURL is the infobox player image (lightmode preferred).
func playerPortraitURL(doc *goquery.Document) *string {
	img := doc.Find(".fo-nttax-infobox .infobox-image-wrapper .infobox-image.lightmode img").First()
	if img.Length() == 0 {
		img = doc.Find(".fo-nttax-infobox .infobox-image-wrapper img").First()
	}
	if img.Length() == 0 {
		return nil
	}
	src := strings.TrimSpace(img.AttrOr("src", ""))
	return profileURL(src)
}

func splitCommaList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		id := cleanText(part)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, id)
	}
	return out
}
