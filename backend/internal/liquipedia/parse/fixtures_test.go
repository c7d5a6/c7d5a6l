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

func documentFromHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	return doc
}
