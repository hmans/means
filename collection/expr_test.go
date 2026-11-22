package collection

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeExprDocs(t *testing.T) *Collection {
	t.Helper()
	c, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"a": "status: todo\n  n: 1", "b": "status: in_progress\n  n: 2\n  tags: [x]",
		"c": "status: done\n  n: 3", "d": "status: todo\n  n: 4",
	} {
		doc := "---\nid: " + name + "\ntitle: Event " + name + "\ndata:\n  " + data + "\n---\nbody of " + name + "\n"
		if err := os.WriteFile(filepath.Join(c.Path(), name+".md"), []byte(doc), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(c.Path(), "broken.md"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	return c
}

func hitIDs(hits []SearchHit) string {
	ids := make([]string, len(hits))
	for i, hit := range hits {
		ids[i] = hit.ID
	}
	return strings.Join(ids, ",")
}

func TestExpressionMatch(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	fm := map[string]any{"id": "abc12", "title": "Test", "data": map[string]any{"n": json.Number("9007199254740993"), "f": json.Number("1.5")}}
	for source, want := range map[string]bool{
		"":                                  true,
		"id == 'abc12' and title == 'Test'": true,
		"body contains 'hello'":             true,
		"data.n == 9007199254740993":        true,
		"data.f > 1":                        true,
		"data.missing == nil":               true,
		"now().Year() == 2026":              true,
		"'x' in (data.tags ?? [])":          false,
	} {
		expr, err := CompileExpression(source)
		if err != nil {
			t.Fatalf("%q: %v", source, err)
		}
		if got, err := expr.Match(fm, "hello", now); err != nil || got != want {
			t.Errorf("%q: got %v err %v", source, got, err)
		}
	}
	for _, source := range []string{"data.status ==", "1 + 1", "'text'"} {
		if _, err := CompileExpression(source); err == nil {
			t.Errorf("compiled %q", source)
		}
	}
	expr, _ := CompileExpression("data.f")
	if _, err := expr.Match(fm, "", now); err == nil {
		t.Error("non-boolean result did not fail")
	}
	expr, _ = CompileExpression("data.n > 0")
	if _, err := expr.Match(map[string]any{"data": map[string]any{"n": json.Number("9223372036854775808")}}, "", now); err == nil {
		t.Error("oversized integer was silently rounded")
	}
}

func TestWhereListAndSearchModes(t *testing.T) {
	c := writeExprDocs(t)
	ctx := context.Background()
	where := `data.status in ["todo", "in_progress"] and data.n > 1`
	listed, err := c.List(ctx, ListOptions{Where: where, Limit: 1})
	if err != nil || len(listed.Documents) != 1 || listed.Documents[0].ID != "b" || !listed.HasMore || len(listed.Diagnostics) != 1 {
		t.Fatalf("list: %+v %v", listed, err)
	}
	listed, err = c.List(ctx, ListOptions{Where: where, Offset: 1, Limit: 1})
	if err != nil || len(listed.Documents) != 1 || listed.Documents[0].ID != "d" || listed.HasMore {
		t.Fatalf("list offset: %+v %v", listed, err)
	}
	listed, err = c.List(ctx, ListOptions{Where: where, Filter: map[string]any{"status": "todo"}})
	if err != nil || len(listed.Documents) != 1 || listed.Documents[0].ID != "d" {
		t.Fatalf("list AND filter: %+v %v", listed, err)
	}
	for _, mode := range []string{"data", "literal", "fulltext"} {
		opts := SearchOptions{Where: where + ` and body contains "of"`}
		if mode != "data" {
			opts.Query = "event"
		}
		opts.Literal = mode == "literal"
		result, err := c.Search(ctx, opts)
		if err != nil || hitIDs(result.Documents) != "b,d" || len(result.Diagnostics) != 1 {
			t.Fatalf("%s: %+v %v", mode, result, err)
		}
		opts.Limit = 1
		result, err = c.Search(ctx, opts)
		if err != nil || hitIDs(result.Documents) != "b" || !result.HasMore {
			t.Fatalf("%s limit: %+v %v", mode, result, err)
		}
		opts.Where = "data.n > 'x'"
		if _, err := c.Search(ctx, opts); err == nil || !strings.Contains(err.Error(), "document a") {
			t.Fatalf("%s runtime error: %v", mode, err)
		}
	}
	for _, where := range []string{"data.status ==", "42"} {
		if _, err := c.List(ctx, ListOptions{Where: where}); err == nil || !strings.Contains(err.Error(), "invalid where expression") {
			t.Errorf("list accepted %q: %v", where, err)
		}
	}
	if _, err := c.List(ctx, ListOptions{Where: "data.status"}); err == nil || !strings.Contains(err.Error(), "expected boolean") {
		t.Errorf("non-boolean result: %v", err)
	}
}

func TestWhereSemanticSearch(t *testing.T) {
	fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	if _, err := c.create(context.Background(), map[string]any{"title": "apple", "data": map[string]any{"kind": "fruit"}}, "orchard"); err != nil {
		t.Fatal(err)
	}
	engine, err := c.create(context.Background(), map[string]any{"title": "engine", "data": map[string]any{"kind": "machine"}}, "machine")
	if err != nil {
		t.Fatal(err)
	}
	all := -1.0
	result, err := c.Search(context.Background(), SearchOptions{Query: "fruit", Semantic: true, MinScore: &all, Where: `data.kind == "machine"`})
	if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != engine.ID {
		t.Fatalf("semantic: %+v %v", result, err)
	}
}
