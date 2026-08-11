package parse_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
)

type fixture struct {
	name     string
	htmlPath string
	metaPath string
	html     string
	meta     liquipedia.FixtureMeta
}

func testdataRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "liquipedia"))
	return root
}

func loadFixtures(t *testing.T) []fixture {
	t.Helper()
	root := testdataRoot(t)

	var out []fixture
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		html, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		metaPath := strings.TrimSuffix(path, ".html") + ".meta.json"
		metaRaw, err := os.ReadFile(metaPath)
		if err != nil {
			return err
		}
		var meta liquipedia.FixtureMeta
		if err := json.Unmarshal(metaRaw, &meta); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, fixture{
			name:     filepath.ToSlash(rel),
			htmlPath: path,
			metaPath: metaPath,
			html:     string(html),
			meta:     meta,
		})
		return nil
	})
	if err != nil {
		t.Fatalf("load fixtures: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("no html fixtures found")
	}
	return out
}

// loadFixturesNamed returns only the listed fixture paths (slash-separated under testdata/liquipedia).
// Extra downloaded HTML (player/team pages, etc.) can sit on disk without joining tournament parse tests.
func loadFixturesNamed(t *testing.T, names map[string]struct{}) []fixture {
	t.Helper()
	all := loadFixtures(t)
	byName := make(map[string]fixture, len(all))
	for _, fx := range all {
		byName[fx.name] = fx
	}
	out := make([]fixture, 0, len(names))
	for name := range names {
		fx, ok := byName[name]
		if !ok {
			t.Fatalf("missing fixture file %s", name)
		}
		out = append(out, fx)
	}
	return out
}

func loadFixturesForExpectations[T any](t *testing.T, want map[string]T) []fixture {
	t.Helper()
	names := make(map[string]struct{}, len(want))
	for k := range want {
		names[k] = struct{}{}
	}
	return loadFixturesNamed(t, names)
}

func documentFromHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	return doc
}
