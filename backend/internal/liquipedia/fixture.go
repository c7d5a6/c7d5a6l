package liquipedia

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FixtureMeta is stored beside each downloaded HTML fixture.
type FixtureMeta struct {
	SourceURL    string    `json:"sourceUrl"`
	Wiki         string    `json:"wiki"`
	Title        string    `json:"title"`
	DisplayTitle string    `json:"displayTitle"`
	FetchedAt    time.Time `json:"fetchedAt"`
	API          string    `json:"api"`
}

// FixturePaths returns the HTML and meta paths for a page under root.
func FixturePaths(root string, ref PageRef) (htmlPath, metaPath string) {
	rel := filepath.Join(append([]string{ref.Wiki}, strings.Split(ref.Title, "/")...)...)
	base := filepath.Join(root, rel)
	return base + ".html", base + ".meta.json"
}

// WriteFixture writes page HTML and metadata under root.
func WriteFixture(root string, page *Page) (htmlPath string, err error) {
	htmlPath, metaPath := FixturePaths(root, page.Ref)
	if err := os.MkdirAll(filepath.Dir(htmlPath), 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(htmlPath, []byte(page.HTML), 0o644); err != nil {
		return "", fmt.Errorf("write html fixture: %w", err)
	}

	meta := FixtureMeta{
		SourceURL:    page.Ref.URL,
		Wiki:         page.Ref.Wiki,
		Title:        page.Title,
		DisplayTitle: page.DisplayTitle,
		FetchedAt:    page.FetchedAt,
		API:          "action=parse&prop=text|displaytitle",
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(metaPath, raw, 0o644); err != nil {
		return "", fmt.Errorf("write meta fixture: %w", err)
	}
	return htmlPath, nil
}
