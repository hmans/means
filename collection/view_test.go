package collection

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeViewDocs(t *testing.T) *Collection {
	t.Helper()
	c := writeExprDocs(t)
	config := "---\nschema: {type: object}\napp:\n  views:\n" +
		"    - title: Open work\n      filter: data.status in [\"todo\", \"in_progress\"]\n      columns: [title]\n" +
		"    - title: Everything\n" +
		"    - title: Broken\n      filter: 'data.status =='\n" +
		"    - title: Failing\n      filter: data.n > 'x'\n---\n"
	if err := os.WriteFile(filepath.Join(c.Path(), ConfigFilename), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(c.Path())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestViewListAndSearchModes(t *testing.T) {
	c := writeViewDocs(t)
	ctx := context.Background()
	listed, err := c.List(ctx, ListOptions{View: "Open work", Where: "data.n > 1", Limit: 1})
	if err != nil || len(listed.Documents) != 1 || listed.Documents[0].ID != "b" || !listed.HasMore || len(listed.Diagnostics) != 1 {
		t.Fatalf("list: %+v %v", listed, err)
	}
	listed, err = c.List(ctx, ListOptions{View: "Open work", Filter: map[string]any{"status": "todo"}, Offset: 1})
	if err != nil || len(listed.Documents) != 1 || listed.Documents[0].ID != "d" {
		t.Fatalf("list AND filter: %+v %v", listed, err)
	}
	listed, err = c.List(ctx, ListOptions{View: "Everything"})
	if err != nil || len(listed.Documents) != 4 {
		t.Fatalf("view without filter: %+v %v", listed, err)
	}
	for _, mode := range []string{"data", "literal", "fulltext"} {
		opts := SearchOptions{View: "Open work", Where: "data.n > 1"}
		if mode != "data" {
			opts.Query = "event"
		}
		opts.Literal = mode == "literal"
		result, err := c.Search(ctx, opts)
		if err != nil || hitIDs(result.Documents) != "b,d" {
			t.Fatalf("%s: %+v %v", mode, result, err)
		}
		opts.View = "Failing"
		if _, err := c.Search(ctx, opts); err == nil || !strings.Contains(err.Error(), `view "Failing" filter failed for document`) {
			t.Fatalf("%s runtime error: %v", mode, err)
		}
	}
}

func TestViewErrors(t *testing.T) {
	c := writeViewDocs(t)
	ctx := context.Background()
	_, err := c.List(ctx, ListOptions{View: "open work"})
	if err == nil || !strings.Contains(err.Error(), `unknown view "open work"; available views: "Open work", "Everything", "Broken", "Failing"`) {
		t.Fatalf("unknown view: %v", err)
	}
	if _, err := c.Search(ctx, SearchOptions{View: "Broken"}); err == nil || !strings.Contains(err.Error(), `invalid filter in view "Broken"`) {
		t.Fatalf("compile error: %v", err)
	}
	empty := writeExprDocs(t)
	if _, err := empty.List(ctx, ListOptions{View: "Open work"}); err == nil || !strings.Contains(err.Error(), "defines no app.views") {
		t.Fatalf("no views: %v", err)
	}
}

func TestViewSemanticSearch(t *testing.T) {
	fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	if _, err := c.create(context.Background(), map[string]any{"title": "apple", "data": map[string]any{"kind": "fruit"}}, "orchard"); err != nil {
		t.Fatal(err)
	}
	engine, err := c.create(context.Background(), map[string]any{"title": "engine", "data": map[string]any{"kind": "machine"}}, "machine")
	if err != nil {
		t.Fatal(err)
	}
	c.app = AppConfig{Views: []AppView{{Title: "Machines", Filter: `data.kind == "machine"`}}}
	all := -1.0
	result, err := c.Search(context.Background(), SearchOptions{Query: "fruit", Semantic: true, MinScore: &all, View: "Machines"})
	if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != engine.ID {
		t.Fatalf("semantic: %+v %v", result, err)
	}
}
