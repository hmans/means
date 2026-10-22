package collection

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/gofrs/flock"
)

func TestFullTextRankingAndFilters(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"a.md": "---\nid: body1\ntitle: Notes\ndata: {}\n---\nTimeout during deployment.",
		"b.md": "---\nid: title\ntitle: Deployment timeout\ndata:\n  count: 9007199254740993\n  tags: [a, b]\n---\nNotes.",
		"c.md": "---\nid: split\ntitle: Deployment\ndata: {}\n---\nTimeout.",
		"d.md": "---\nid: other\ntitle: Timeouts\ndata: {}\n---\nDeployment.",
	} {
		if err := os.WriteFile(filepath.Join(c.Path, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := c.Search(SearchOptions{Query: "DEPLOYMENT timeout", Limit: 1})
	if err != nil || len(r.Documents) != 1 || r.Documents[0].ID != "title" || !r.HasMore {
		t.Fatalf("ranking: %+v %v", r, err)
	}
	r, err = c.Search(SearchOptions{Query: "timeout deployment"})
	if err != nil || len(r.Documents) != 3 || r.HasMore {
		t.Fatalf("AND or stemming: %+v %v", r, err)
	}
	for _, hit := range r.Documents {
		if hit.Match == nil || hit.Match.Excerpt == "" {
			t.Fatal("missing excerpt")
		}
	}
	r, err = c.Search(SearchOptions{Query: "deployment timeout", Literal: true})
	if err != nil || len(r.Documents) != 1 || r.Documents[0].ID != "title" {
		t.Fatalf("literal: %+v %v", r, err)
	}
	r, err = c.Search(SearchOptions{Query: "timeout deployment", Filter: map[string]any{"count": json.Number("9007199254740993"), "tags": []any{"a", "b"}}, Limit: 1})
	if err != nil || len(r.Documents) != 1 || r.Documents[0].ID != "title" || r.HasMore {
		t.Fatalf("precise filter: %+v %v", r, err)
	}
	for _, q := range []string{"!!!", "   ", "deployment absent"} {
		r, err = c.Search(SearchOptions{Query: q})
		if err != nil || len(r.Documents) != 0 || r.HasMore {
			t.Fatalf("empty %q: %+v %v", q, r, err)
		}
	}
	if _, err := c.Create(map[string]any{"title": "Über den Fehler 123"}, "the and"); err != nil {
		t.Fatal(err)
	}
	r, err = c.Search(SearchOptions{Query: "ÜBER the 123 and"})
	if err != nil || len(r.Documents) != 1 {
		t.Fatalf("unicode/stop words/numbers: %+v %v", r, err)
	}
}

func TestFullTextReconcilesFilesystem(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(map[string]any{"title": "Alpha"}, "red")
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(c.Path, doc.filename)
	check := func(q string, count, diagnostics int) {
		t.Helper()
		r, err := c.Search(SearchOptions{Query: q})
		if err != nil || len(r.Documents) != count || len(r.Diagnostics) != diagnostics {
			t.Fatalf("%s: %+v %v", q, r, err)
		}
	}
	check("red", 1, 0)
	stat, err := os.Stat(filename)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(doc.Markdown, "red", "tan", 1)
	if err := os.WriteFile(filename, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filename, stat.ModTime(), stat.ModTime()); err != nil {
		t.Fatal(err)
	}
	check("red", 0, 0)
	check("tan", 1, 0)
	if err := os.Mkdir(filepath.Join(c.Path, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(c.Path, "nested", "renamed.md")
	if err := os.Rename(filename, moved); err != nil {
		t.Fatal(err)
	}
	check("tan", 1, 0)
	if err := os.WriteFile(filename, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	check("tan", 0, 2)
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	check("tan", 1, 0)
	boundary := filepath.Join(c.Path, "nested", SchemaFilename)
	if err := os.WriteFile(boundary, []byte("invalid schema"), 0o644); err != nil {
		t.Fatal(err)
	}
	check("tan", 0, 0)
	if err := os.Remove(boundary); err != nil {
		t.Fatal(err)
	}
	check("tan", 1, 0)
	if err := os.WriteFile(moved, []byte("malformed"), 0o644); err != nil {
		t.Fatal(err)
	}
	check("tan", 0, 1)
	if err := os.WriteFile(moved, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	check("tan", 1, 0)
	if err := os.Remove(moved); err != nil {
		t.Fatal(err)
	}
	check("tan", 0, 0)
	// A new document appears without going through means.
	if err := os.WriteFile(filename, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	check("tan", 1, 0)
	cache := filepath.Join(c.Path, ".means", "search")
	if err := os.RemoveAll(cache); err != nil {
		t.Fatal(err)
	}
	check("tan", 1, 0)
	index, err := bleve.Open(cache)
	if err != nil {
		t.Fatal(err)
	}
	if err := index.SetInternal(searchManifestKey, []byte(`{"version":999,"hashes":{}}`)); err != nil {
		t.Fatal(err)
	}
	if err := index.Close(); err != nil {
		t.Fatal(err)
	}
	check("tan", 1, 0)
	if err := os.RemoveAll(cache); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "index_meta.json"), []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	check("tan", 1, 0)
}

func TestFullTextFiltersBeforePaging(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 105; i++ {
		content := fmt.Sprintf("---\nid: doc%03d\ntitle: Needle\ndata:\n  keep: %t\n---\n", i, i >= 103)
		if err := os.WriteFile(filepath.Join(c.Path, fmt.Sprintf("%03d.md", i)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := c.Search(SearchOptions{Query: "needle", Filter: map[string]any{"keep": true}, Limit: 1})
	if err != nil || len(r.Documents) != 1 || r.Documents[0].ID != "doc103" || !r.HasMore {
		t.Fatalf("paging: %+v %v", r, err)
	}
	r, err = c.Search(SearchOptions{Query: "needle", Filter: map[string]any{"keep": true}, Limit: 2})
	if err != nil || len(r.Documents) != 2 || r.HasMore {
		t.Fatalf("paging end: %+v %v", r, err)
	}
}

func TestSearchCacheSafetyAndConcurrency(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Create(map[string]any{"title": "Needle"}, ""); err != nil {
		t.Fatal(err)
	}
	for _, opts := range []SearchOptions{{Query: "needle", Literal: true}, {}} {
		if _, err := c.Search(opts); err != nil {
			t.Fatal(err)
		}
	}
	cache := filepath.Join(c.Path, ".means")
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatal("scan-only search made a cache")
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, cache); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Search(SearchOptions{Query: "needle"}); err == nil {
		t.Fatal("followed cache symlink")
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatal("wrote outside collection", err)
	}
	if err := os.Remove(cache); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 6 {
		wg.Go(func() {
			r, err := c.Search(SearchOptions{Query: "needle"})
			if err != nil || len(r.Documents) != 1 {
				t.Errorf("concurrent search: %+v %v", r, err)
			}
		})
	}
	wg.Wait()
	for range 3 {
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestSearchProcessHelper$")
		cmd.Env = append(os.Environ(), "MEANS_SEARCH_TEST_ROOT="+c.Path)
		wg.Go(func() {
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("process search: %v %s", err, output)
			}
		})
	}
	wg.Wait()
	crash := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestSearchProcessHelper$")
	crash.Env = append(os.Environ(), "MEANS_SEARCH_TEST_ROOT="+c.Path, "MEANS_SEARCH_UNCLEAN_CLOSE=1")
	if output, err := crash.CombinedOutput(); err != nil {
		t.Fatalf("unclean close helper: %v %s", err, output)
	}
	if r, err := c.Search(SearchOptions{Query: "needle"}); err != nil || len(r.Documents) != 1 {
		t.Fatalf("recovery after process exit: %+v %v", r, err)
	}
	lock := flock.New(filepath.Join(cache, "search.lock"))
	if err := lock.Lock(); err != nil {
		t.Fatal(err)
	}
	_, searchErr := c.Search(SearchOptions{Query: "needle"})
	if err := lock.Unlock(); err != nil {
		t.Fatal(err)
	}
	if searchErr == nil {
		t.Fatal("ignored cache lock")
	}
	if _, err := c.Search(SearchOptions{Query: "needle"}); err != nil {
		t.Fatal("lock not released", err)
	}
	// Never follow an embedded cache symlink during open or automatic rebuild.
	link := filepath.Join(cache, "search", "unexpected-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Search(SearchOptions{Query: "needle"}); err == nil {
		t.Fatal("followed embedded cache symlink")
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatal("modified symlink target", err)
	}
}

func TestSearchProcessHelper(t *testing.T) {
	path := os.Getenv("MEANS_SEARCH_TEST_ROOT")
	if path == "" {
		return
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("MEANS_SEARCH_UNCLEAN_CLOSE") == "1" {
		lock := flock.New(filepath.Join(path, ".means", "search.lock"))
		if err := lock.Lock(); err != nil {
			t.Fatal(err)
		}
		m, err := searchMapping()
		if err != nil {
			t.Fatal(err)
		}
		index, _, err := openSearchIndex(filepath.Join(path, ".means", "search"), m)
		if err != nil {
			t.Fatal(err)
		}
		// Simulate exit after a committed metadata write but before closing either
		// the index or the OS lock. The next search must recover and reconcile.
		if err := index.SetInternal([]byte("test-interruption"), []byte("committed")); err != nil {
			t.Fatal(err)
		}
		os.Exit(0)
	}
	r, err := c.Search(SearchOptions{Query: "needle"})
	if err != nil || len(r.Documents) != 1 {
		t.Fatalf("search: %+v %v", r, err)
	}
}

func TestInitGitignore(t *testing.T) {
	for _, original := range []string{"", "# Keep\nother/", ".means/\n", "/.means/\n", "# Keep\r\n"} {
		dir := t.TempDir()
		filename := filepath.Join(dir, ".gitignore")
		if err := os.WriteFile(filename, []byte(original), 0o640); err != nil {
			t.Fatal(err)
		}
		if _, err := Init(dir, nil); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		want := original
		if !strings.Contains(original, ".means/\n") {
			if want != "" && !strings.HasSuffix(want, "\n") {
				want += "\n"
			}
			want += ".means/\n"
		}
		if string(got) != want {
			t.Fatalf("gitignore: %q want %q", got, want)
		}
		if _, err := Init(dir, nil); err == nil {
			t.Fatal("reinit succeeded")
		}
		after, _ := os.ReadFile(filename)
		if string(after) != want {
			t.Fatal("reinit changed gitignore")
		}
	}
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "ignore")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(dir, nil); err == nil {
		t.Fatal("followed gitignore symlink")
	}
	if _, err := os.Stat(filepath.Join(dir, SchemaFilename)); !os.IsNotExist(err) {
		t.Fatal("failed init created schema")
	}
}

// Baseline compares warm persistence with building the same text mapping in memory.
func BenchmarkFullTextIndex(b *testing.B) {
	for _, count := range []int{100, 1000} {
		c, err := Init(b.TempDir(), nil)
		if err != nil {
			b.Fatal(err)
		}
		for i := 0; i < count; i++ {
			content := fmt.Sprintf("---\nid: doc%d\ntitle: Deployment notes %d\ndata: {}\n---\n%s", i, i, strings.Repeat("A timeout during deployment needs investigation. ", 40))
			if err := os.WriteFile(filepath.Join(c.Path, fmt.Sprintf("%d.md", i)), []byte(content), 0o644); err != nil {
				b.Fatal(err)
			}
		}
		if _, err := c.Search(SearchOptions{Query: "timeout", Limit: 20}); err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("warm/%d", count), func(b *testing.B) {
			for b.Loop() {
				if _, err := c.Search(SearchOptions{Query: "timeout", Limit: 20}); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("memory/%d", count), func(b *testing.B) {
			for b.Loop() {
				root, err := os.OpenRoot(c.Path)
				if err != nil {
					b.Fatal(err)
				}
				entries, err := scanDocuments(root)
				root.Close()
				if err != nil {
					b.Fatal(err)
				}
				m, err := searchMapping()
				if err != nil {
					b.Fatal(err)
				}
				index, err := bleve.NewMemOnly(m)
				if err != nil {
					b.Fatal(err)
				}
				batch := index.NewBatch()
				for _, entry := range entries {
					if err := batch.Index(entry.doc.ID, map[string]any{"title": entry.doc.Frontmatter["title"], "body": entry.doc.Body}); err != nil {
						b.Fatal(err)
					}
				}
				if err := index.Batch(batch); err != nil {
					b.Fatal(err)
				}
				q := bleve.NewTermQuery("timeout")
				q.SetField("body")
				if _, err := index.Search(bleve.NewSearchRequestOptions(q, 20, 0, false)); err != nil {
					b.Fatal(err)
				}
				if err := index.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
