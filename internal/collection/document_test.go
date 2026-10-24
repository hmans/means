package collection

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDocumentRoundTrip(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	frontmatter := map[string]any{
		"strings":  []any{"true", "null", "123", "2026-10-20", "yes", "on", "", "hello\nworld"},
		"integer":  json.Number("9007199254740993"),
		"huge":     json.Number("18446744073709551616"),
		"decimal":  json.Number("1.234567890123456789"),
		"exponent": json.Number("1e+30"),
		"nested":   map[string]any{"enabled": true, "nothing": nil, "empty": map[string]any{}},
		"date":     "2026-10-20",
		"<<":       "<<",
	}
	for _, body := range []string{"", "No trailing newline", "# Notes\n\nText\n", "---\nbody divider\n", "Line one\r\nLine two\r\n", "こんにちは 🌱\n"} {
		t.Run(body, func(t *testing.T) {
			doc, err := c.Create(map[string]any{"title": "Round trip", "data": frontmatter}, body)
			if err != nil {
				t.Fatal(err)
			}
			if len(doc.ID) != 5 || !documentID.MatchString(doc.ID) {
				t.Fatalf("unexpected ID %q", doc.ID)
			}
			read, err := c.Read(doc.ID)
			if err != nil {
				t.Fatalf("Read() = %v\n%s", err, doc.Markdown)
			}
			want, _ := json.Marshal(frontmatter)
			got, _ := json.Marshal(read.Frontmatter["data"])
			if !bytes.Equal(got, want) {
				t.Errorf("frontmatter changed\ngot: %s\nwant: %s\n%s", got, want, doc.Markdown)
			}
			if read.Body != body || read.Markdown != doc.Markdown {
				t.Errorf("body or Markdown changed: %q", read.Body)
			}
		})
	}
}

func TestCreateValidationDoesNotWrite(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, metadata := range []map[string]any{nil, {}, {"title": true}, {"title": "ok", "unknown": true}, {"title": make(chan int)}} {
		if _, err := c.Create(metadata, "body"); err == nil {
			t.Fatalf("invalid frontmatter accepted: %v", metadata)
		}
	}
	entries, err := os.ReadDir(c.Path)
	if err != nil || len(entries) != 2 || entries[0].Name() != ".gitignore" || entries[1].Name() != ConfigFilename {
		t.Fatalf("invalid writes changed collection: %v, %v", entries, err)
	}
}

func TestReadManualDocumentAndSchemaViolation(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	markdown := "---\r\nid: manual-note\r\ntitle: Manual\r\ndata:\r\n  summary: retained after a schema change\r\n  date: 2026-10-20\r\n  count: 0x10\r\n---\r\n# Body\r\n"
	if err := os.WriteFile(filepath.Join(c.Path, "manual-note.md"), []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := c.Read("manual-note")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Markdown != markdown || doc.Body != "# Body\r\n" || doc.Frontmatter["data"].(map[string]any)["date"] != "2026-10-20" || doc.Frontmatter["data"].(map[string]any)["count"] != json.Number("16") {
		t.Fatalf("unexpected document: %#v", doc)
	}
	if err := c.Validate(doc.Frontmatter); err == nil {
		t.Fatal("test document should violate the schema but remain readable")
	}
}

func TestMalformedMarkdown(t *testing.T) {
	for _, input := range []string{
		"", "body only", "---\ntitle: missing closing delimiter", "---\n---\nbody",
		"---\n# comment only\n---\n", "---\nnull\n---\n", "---\n[]\n---\n",
		"---\ntitle: [broken\n---\n", "---\ntitle: one\ntitle: two\n---\n",
		"---\n123: value\n---\n", "---\nvalue: .inf\n---\n", "---\nvalue: .nan\n---\n",
		"---\nvalue: &value example\ncopy: *value\n---\n",
		"---\nvalue: !custom text\n---\n", "---\n!custom {}\n---\n",
		"---\nvalue: !custom []\n---\n", "---\nvalue: 1\n...\nextra\n---\n",
		"---\ntitle: \xff\n---\n",
	} {
		t.Run(input, func(t *testing.T) {
			if _, _, err := parseMarkdown([]byte(input)); err == nil {
				t.Fatalf("malformed Markdown accepted: %q", input)
			}
		})
	}
}

func TestListLimitsAndMalformedFiles(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	empty, err := c.List(20)
	if err != nil || empty.Documents == nil || empty.Diagnostics == nil || len(empty.Documents) != 0 || empty.HasMore {
		t.Fatalf("unexpected empty listing: %#v, %v", empty, err)
	}
	for name, content := range map[string]string{
		"b.md": "malformed", "a.md": "---\nid: alpha\ntitle: A\ndata: {}\n---\nbody", "c.md": "---\nid: charlie\ntitle: C\ndata: {}\n---\nbody",
		"other.txt": "ignored", ".hidden.md": "ignored",
		"MEANS.md": "# Collection workflow\nThis is configuration, not a document.",
	} {
		if err := os.WriteFile(filepath.Join(c.Path, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(c.Path, "directory.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		limit, count int
		more         bool
	}{{1, 1, true}, {2, 2, false}, {3, 2, false}, {20, 2, false}, {0, 2, false}} {
		result, err := c.List(tt.limit)
		if err != nil || len(result.Documents) != tt.count || result.HasMore != tt.more {
			t.Fatalf("List(%d) = %#v, %v", tt.limit, result, err)
		}
		ids := []string{}
		for _, doc := range result.Documents {
			ids = append(ids, doc.ID)
		}
		if !reflect.DeepEqual(ids, []string{"alpha", "charlie"}[:tt.count]) {
			t.Fatalf("unexpected order: %v", ids)
		}
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Path != "b.md" || result.Diagnostics[0].Error == "" {
			t.Fatal("malformed document was not reported")
		}
		data, _ := json.Marshal(result.Documents[0])
		if !strings.Contains(string(data), `"data":{}`) || strings.Contains(string(data), "body") {
			t.Fatalf("unexpected summary: %s", data)
		}
	}
	if _, err := c.List(-1); err == nil {
		t.Fatal("negative limit accepted")
	}
}

func TestReadRejectsPathsAndSymlinks(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", ".", "..", "../outside", "a/b", `a\b`, "/tmp/outside", "note.md"} {
		if _, err := c.Read(id); err == nil {
			t.Fatalf("unsafe ID accepted: %q", id)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("---\ntitle: outside\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(c.Path, "linked.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := c.Read("linked"); err == nil {
		t.Fatal("symlink accepted")
	}
	result, err := c.List(0)
	if err != nil || len(result.Documents) != 0 {
		t.Fatalf("symlink listed: %#v, %v", result, err)
	}
}

func TestListDiagnosticsMatchSearchRegardlessOfLimit(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`false`))
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"a.md": "broken before",
		"b.md": "---\nid: valid\ntitle: Valid envelope\ndata: {}\n---\nbody",
		"c.md": "---\nid: duplicate\ntitle: First\ndata: {}\n---\nbody",
		"d.md": "---\nid: duplicate\ntitle: Second\ndata: {}\n---\nbody",
		"z.md": "broken after",
	} {
		if err := os.WriteFile(filepath.Join(c.Path, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, limit := range []int{1, 2, 0} {
		list, err := c.ListWithOptions(ListOptions{Limit: limit, WithBody: true})
		if err != nil {
			t.Fatal(err)
		}
		search, err := c.Search(SearchOptions{Limit: limit})
		if err != nil {
			t.Fatal(err)
		}
		if len(list.Documents) != 1 || list.Documents[0].ID != "valid" || list.Documents[0].Body == nil || *list.Documents[0].Body != "body" || list.HasMore {
			t.Fatalf("limit %d: %+v", limit, list)
		}
		if len(list.Diagnostics) != 4 || !reflect.DeepEqual(list.Diagnostics, search.Diagnostics) {
			t.Fatalf("diagnostics differ: %+v / %+v", list, search)
		}
	}
	if err := os.Remove(filepath.Join(c.Path, "b.md")); err != nil {
		t.Fatal(err)
	}
	list, err := c.List(1)
	if err != nil || len(list.Documents) != 0 || len(list.Diagnostics) != 4 || list.HasMore {
		t.Fatalf("all invalid: %+v %v", list, err)
	}
}
