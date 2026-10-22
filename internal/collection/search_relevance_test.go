package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFullTextExcerptCoverage(t *testing.T) {
	m, err := searchMapping()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, title, body, query, field string
		contains                        []string
	}{
		{"body covers query", "Search improvements", "The search index stores document hashes.", "search index", "body", []string{"search", "index"}},
		{"late passage", "Search", "search " + strings.Repeat("padding ", 50) + "The search index catches edits.", "search index", "body", []string{"search", "index"}},
		{"repetition cannot win", "Search index", "search search search search search", "search index", "title", []string{"Search", "index"}},
		{"title wins tie", "Search index", "Search index", "search index", "title", []string{"Search", "index"}},
		{"split fields", "Deployment", "Timeout", "deployment timeout", "title", []string{"Deployment"}},
		{"unicode", "Notes", strings.Repeat("猫 ", 100) + "Über den Fehler im Suchindex.", "ÜBER suchindex", "body", []string{"Über", "Suchindex"}},
		{"term at window edge", "Index", "search " + strings.Repeat("x", 168) + " index", "search index", "title", []string{"Index"}},
		{"wide passage", "Search", strings.Repeat("padding ", 20) + "search " + strings.Repeat("x", 150) + " index", "search index", "body", []string{"search", "index"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := m.AnalyzeText(searchAnalyzer, []byte(tt.query))
			if err != nil {
				t.Fatal(err)
			}
			terms := map[string]bool{}
			for _, token := range tokens {
				terms[string(token.Term)] = true
			}
			match, err := fullTextExcerpt(m, &Document{Frontmatter: map[string]any{"title": tt.title}, Body: tt.body}, terms)
			if err != nil || match == nil || match.Field != tt.field {
				t.Fatalf("excerpt: %+v %v", match, err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(match.Excerpt, want) {
					t.Fatalf("missing %q in %q", want, match.Excerpt)
				}
			}
			if !utf8.ValidString(match.Excerpt) || utf8.RuneCountInString(match.Excerpt) > 182 {
				t.Fatalf("unbounded/invalid excerpt: %q", match.Excerpt)
			}
		})
	}
}

// Fixed issue-like documents and expected first results make ranking changes
// reviewable without relying on the repository's changing issue contents.
func TestSearchRelevanceFixture(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	for _, doc := range []struct{ id, title, body string }{
		{"index", "Search index refresh", "Reconcile the search index with changed Markdown files."},
		{"schema", "Schema validation", "Validate a replacement schema before installation."},
		{"delete", "Delete documents", "Remove documents using their current content hash."},
		{"deploy", "Deployment timeout", "Investigate a timeout during deployment."},
		{"notes", "Project notes", "Discuss search index refresh, schema validation, delete documents, and deployment timeout."},
	} {
		content := fmt.Sprintf("---\nid: %s\ntitle: %s\ndata: {}\n---\n%s", doc.id, doc.title, doc.body)
		if err := os.WriteFile(filepath.Join(c.Path, doc.id+".md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range []struct{ query, id string }{{"search index", "index"}, {"validation schema", "schema"}, {"delete documents", "delete"}, {"timeout deployment", "deploy"}} {
		t.Run(tt.query, func(t *testing.T) {
			for range 2 { // Cold/reconciled and reused index must agree.
				r, err := c.Search(SearchOptions{Query: tt.query, Limit: 1})
				if err != nil || len(r.Documents) != 1 || r.Documents[0].ID != tt.id || !r.HasMore {
					t.Fatalf("ranking: %+v %v", r, err)
				}
				for _, term := range strings.Fields(tt.query) {
					if !strings.Contains(strings.ToLower(r.Documents[0].Match.Excerpt), term) {
						t.Fatalf("missing %q: %+v", term, r.Documents[0].Match)
					}
				}
			}
		})
	}
}

// Compare parsing with a persisted parse cache and read+hash work alone.
// The no-parse baseline is only a lower bound: it excludes cache decoding,
// envelope checks, duplicate detection, and cache invalidation costs.
func BenchmarkSearchScanCosts(b *testing.B) {
	c, err := Init(b.TempDir(), nil)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		content := fmt.Sprintf("---\nid: doc%d\ntitle: Search notes\ndata:\n  tags: [search, index]\n---\n%s", i, strings.Repeat("Document body with search terms. ", 60))
		if err := os.WriteFile(filepath.Join(c.Path, fmt.Sprintf("%d.md", i)), []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	b.Run("read_hash_parse", func(b *testing.B) {
		for b.Loop() {
			root, err := os.OpenRoot(c.Path)
			if err != nil {
				b.Fatal(err)
			}
			entries, err := scanDocuments(root)
			root.Close()
			if err != nil || len(entries) != 1000 {
				b.Fatalf("scan: %d %v", len(entries), err)
			}
		}
	})
	b.Run("read_hash_only_lower_bound", func(b *testing.B) {
		for b.Loop() {
			root, err := os.OpenRoot(c.Path)
			if err != nil {
				b.Fatal(err)
			}
			entries, err := os.ReadDir(c.Path)
			if err != nil {
				b.Fatal(err)
			}
			for _, entry := range entries {
				if !strings.HasSuffix(entry.Name(), ".md") {
					continue
				}
				if _, err := root.Lstat(entry.Name()); err != nil {
					b.Fatal(err)
				}
				data, err := root.ReadFile(entry.Name())
				if err != nil {
					b.Fatal(err)
				}
				if len(contentHash(string(data))) != 64 {
					b.Fatal("hash failed")
				}
			}
			root.Close()
		}
	})
	b.Run("read_hash_cached_metadata", func(b *testing.B) {
		root, err := os.OpenRoot(c.Path)
		if err != nil {
			b.Fatal(err)
		}
		defer root.Close()
		cache := filepath.Join(b.TempDir(), "documents.jsonl")
		if _, err := scanCachedDocuments(root, cache); err != nil {
			b.Fatal(err)
		}
		for b.Loop() {
			entries, err := scanCachedDocuments(root, cache)
			if err != nil || len(entries) != 1000 {
				b.Fatalf("cached scan: %d %v", len(entries), err)
			}
		}
	})
}
