package collection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorReportsInvalidDocumentsWithoutWriting(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{"type":"object","properties":{"count":{"type":"integer"}},"required":["count"],"additionalProperties":false}`))
	if err != nil {
		t.Fatal(err)
	}
	initial, err := c.Doctor()
	if err != nil || !initial.Healthy || initial.Checked != 0 || initial.Diagnostics == nil || len(initial.Diagnostics) != 0 {
		t.Fatalf("empty Doctor() = %#v, %v", initial, err)
	}
	if _, err := c.Create(map[string]any{"title": "Created", "data": map[string]any{"count": 1}}, "body"); err != nil {
		t.Fatal(err)
	}
	healthy, err := c.Doctor()
	if err != nil || !healthy.Healthy || healthy.Checked != 1 || len(healthy.Diagnostics) != 0 {
		t.Fatalf("healthy Doctor() = %#v, %v", healthy, err)
	}
	if err := os.Mkdir(filepath.Join(c.Path, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"healthy.md":        "---\nid: healthy\ntitle: Healthy\ndata:\n  count: 1\n---\nbody",
		"nested/invalid.md": "---\nid: invalid\ntitle: Invalid\ndata:\n  count: wrong\n---\nbody",
		"malformed.md":      "---\nid: malformed\ntitle: [broken\n---\n",
		"envelope.md":       "---\nid: envelope\ntitle: ''\ndata: {}\n---\n",
		"first.md":          "---\nid: repeated\ntitle: First\ndata:\n  count: 1\n---\n",
		"second.md":         "---\nid: repeated\ntitle: Second\ndata:\n  count: 2\n---\n",
		".hidden.md":        "broken",
		"MEANS.md":          "broken",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(c.Path, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before := map[string]string{}
	for name := range files {
		data, err := os.ReadFile(filepath.Join(c.Path, name))
		if err != nil {
			t.Fatal(err)
		}
		before[name] = string(data)
	}
	result, err := c.Doctor()
	if err != nil {
		t.Fatal(err)
	}
	if result.Healthy || result.Checked != 7 || len(result.Diagnostics) != 5 {
		t.Fatalf("unexpected report: %#v", result)
	}
	want := map[string]string{
		"envelope.md":       "frontmatter.title",
		"first.md":          "duplicate document ID",
		"malformed.md":      "parse YAML frontmatter",
		"nested/invalid.md": "frontmatter.data does not satisfy the schema",
		"second.md":         "duplicate document ID",
	}
	for _, d := range result.Diagnostics {
		if !strings.Contains(d.Error, want[d.Path]) || want[d.Path] == "" {
			t.Errorf("unexpected diagnostic: %#v", d)
		}
		if d.Path == "nested/invalid.md" && (d.ID != "invalid" || !strings.Contains(d.Error, "count")) {
			t.Errorf("schema error lacks ID or field: %#v", d)
		}
		delete(want, d.Path)
	}
	if len(want) != 0 {
		t.Errorf("missing diagnostics: %v", want)
	}
	for name, content := range before {
		data, err := os.ReadFile(filepath.Join(c.Path, name))
		if err != nil || string(data) != content {
			t.Errorf("doctor changed %s: %v", name, err)
		}
	}
}
