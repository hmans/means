package collection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchTextFiltersLimitsAndDiagnostics(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{"type":"object","properties":{"type":{"type":"string"},"priority":{"type":"string"},"count":{"type":"integer"},"details":{"type":"object"}},"required":["type","priority"],"additionalProperties":false}`))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"a.md":       "---\nid: alpha\ntitle: Schema Guide\ndata:\n  type: feature\n  priority: normal\n  count: 9007199254740993\n  details:\n    owner: Mira\n    state: open\n---\nFirst body",
		"b.md":       "---\nid: beta\ntitle: Other\ndata:\n  type: feature\n  priority: high\n---\nThe SCHEMA is evolving.",
		"c.md":       "---\nid: gamma\ntitle: Search\ndata:\n  type: task\n  priority: normal\n---\nA schema (literal) in the body.",
		"d.md":       "---\nid: stale\ntitle: Schema after evolution\ndata:\n  type: feature\n---\nStill searchable",
		"e.md":       "---\nid: duplicate\ntitle: Schema one\ndata:\n  type: feature\n  priority: normal\n---\n",
		"f.md":       "---\nid: duplicate\ntitle: Schema two\ndata:\n  type: feature\n  priority: normal\n---\n",
		"g.md":       "broken",
		".hidden.md": "broken",
		"MEANS.md":   "broken",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(c.Path, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := c.Search(SearchOptions{Query: "schema", Limit: 2, Literal: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Documents) != 2 || result.Documents[0].ID != "alpha" || result.Documents[1].ID != "beta" || !result.HasMore || len(result.Diagnostics) != 3 {
		t.Fatalf("limited search = %#v", result)
	}
	if result.Documents[0].Match.Field != "title" || result.Documents[1].Match.Field != "body" || !strings.Contains(result.Documents[1].Match.Excerpt, "SCHEMA") {
		t.Fatalf("missing useful match context: %#v", result.Documents)
	}
	for _, d := range result.Diagnostics {
		if d.Path == "e.md" || d.Path == "f.md" {
			if d.ID != "duplicate" || !strings.Contains(d.Error, "duplicate") {
				t.Errorf("bad duplicate diagnostic: %#v", d)
			}
		} else if d.Path != "g.md" || d.ID != "" {
			t.Errorf("bad malformed diagnostic: %#v", d)
		}
	}
	result, err = c.Search(SearchOptions{Query: "SCHEMA", Limit: 0, Literal: true})
	if err != nil || len(result.Documents) != 4 || result.HasMore || result.Documents[3].ID != "stale" {
		t.Fatalf("unlimited search, including schema-invalid data = %#v, %v", result, err)
	}
	result, err = c.Search(SearchOptions{Filter: map[string]any{"type": "feature", "priority": "normal"}, Limit: 0})
	if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != "alpha" || result.Documents[0].Match != nil {
		t.Fatalf("filter-only search = %#v, %v", result, err)
	}
	result, err = c.Search(SearchOptions{Filter: map[string]any{"count": json.Number("9007199254740993"), "details": map[string]any{"owner": "Mira"}}, Limit: 0})
	if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != "alpha" {
		t.Fatalf("typed nested filter = %#v, %v", result, err)
	}
	result, err = c.Search(SearchOptions{Query: "schema (literal)", Limit: 0, Literal: true})
	if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != "gamma" {
		t.Fatalf("literal phrase search = %#v, %v", result, err)
	}
	result, err = c.Search(SearchOptions{Filter: map[string]any{"priority": nil}, Limit: 0})
	if err != nil || len(result.Documents) != 0 || result.Documents == nil || result.HasMore || len(result.Diagnostics) != 3 {
		t.Fatalf("empty search = %#v, %v", result, err)
	}
	if _, err := c.Search(SearchOptions{Limit: -1}); err == nil {
		t.Fatal("negative limit accepted")
	}
}
