package collection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestUpdateMergePatch(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{"type":"object","required":["required"],"properties":{"required":{"type":"string"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	original, err := c.Create(map[string]any{"title": "Before", "data": map[string]any{
		"required": "Keep", "remove": "old", "keep": true, "tags": []any{"old", "other"},
		"nested": map[string]any{"keep": "yes", "remove": "old", "change": "before"},
		"scalar": "replace with object", "count": json.Number("9007199254740993"),
	}}, "# Original\r\n\r\nKeep body bytes.\r\n")
	if err != nil {
		t.Fatal(err)
	}
	patch := map[string]any{
		"required": "Keep", "remove": nil, "tags": []any{"new"},
		"nested": map[string]any{"remove": nil, "change": "after"},
		"scalar": map[string]any{"new": "value", "absent": nil},
		"ratio":  json.Number("0.1234567890123456789"),
	}
	before, _ := json.Marshal(patch)
	updated, err := c.Update(original.ID, original.Hash, Patch{Frontmatter: map[string]any{"title": "After", "data": patch}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != original.ID || updated.Hash == original.Hash || updated.Body != original.Body {
		t.Fatalf("unexpected update: %+v", updated)
	}
	want := map[string]any{
		"required": "Keep", "keep": true, "tags": []any{"new"}, "count": json.Number("9007199254740993"),
		"nested": map[string]any{"keep": "yes", "change": "after"},
		"scalar": map[string]any{"new": "value"}, "ratio": json.Number("0.1234567890123456789"),
	}
	if !reflect.DeepEqual(updated.Frontmatter["data"], want) {
		t.Fatalf("metadata: %#v", updated.Frontmatter)
	}
	after, _ := json.Marshal(patch)
	if string(before) != string(after) || original.Frontmatter["title"] != "Before" {
		t.Fatal("patch or original was mutated")
	}
	read, err := c.Read(updated.ID)
	if err != nil || !reflect.DeepEqual(read, updated) {
		t.Fatalf("stored result mismatch: %+v %v", read, err)
	}
	assertNoUpdateArtifacts(t, c.Path)
}

func TestUpdateExactHashBodyAndNoop(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(c.Path, "manual.md")
	for _, original := range []string{
		"---\r\n# keep this comment\r\nid: manual\r\ndata: {}\r\ntitle: 'Manual'\r\n---\r\nOld body\r\n",
		"---\nid: manual\ndata: {}\ntitle: Manual\n---",
	} {
		if err := os.WriteFile(filename, []byte(original), 0o640); err != nil {
			t.Fatal(err)
		}
		read, err := c.Read("manual")
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(original))
		if read.Hash != hex.EncodeToString(sum[:]) {
			t.Fatal("hash is not of exact file bytes")
		}
		noop, err := c.Update("manual", read.Hash, Patch{Frontmatter: map[string]any{}})
		if err != nil || noop.Hash != read.Hash || noop.Markdown != original {
			t.Fatalf("no-op rewrote file: %+v %v", noop, err)
		}
		body := "# Replaced\n\nExact body.\n"
		updated, err := c.Update("manual", read.Hash, Patch{Body: &body})
		if err != nil {
			t.Fatal(err)
		}
		prefix := original[:len(original)-len(read.Body)]
		if !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}
		if updated.Markdown != prefix+body {
			t.Fatalf("body-only update changed header: %q", updated.Markdown)
		}
		stat, err := os.Stat(filename)
		if err != nil || stat.Mode().Perm() != 0o640 {
			t.Fatalf("permissions changed: %v %v", stat, err)
		}
		empty := ""
		cleared, err := c.Update("manual", updated.Hash, Patch{Body: &empty})
		if err != nil || cleared.Body != "" {
			t.Fatalf("clear body: %+v %v", cleared, err)
		}
	}
	assertNoUpdateArtifacts(t, c.Path)
}

func TestUpdateFailuresPreserveOriginal(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(map[string]any{"title": "Keep"}, "Original\n")
	if err != nil {
		t.Fatal(err)
	}
	body := "Changed"
	invalidBody := "\xff"
	for _, tt := range []struct {
		name, id, hash string
		patch          Patch
	}{
		{"missing hash", doc.ID, "", Patch{Body: &body}},
		{"invalid hash", doc.ID, "abc", Patch{Body: &body}},
		{"stale hash", doc.ID, strings.Repeat("0", 64), Patch{Body: &body}},
		{"empty patch", doc.ID, doc.Hash, Patch{}},
		{"invalid body", doc.ID, doc.Hash, Patch{Body: &invalidBody}},
		{"invalid body with metadata", doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"title": "Changed"}, Body: &invalidBody}},
		{"required deletion", doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"title": nil}}},
		{"wrong type", doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"title": 42}}},
		{"unknown field", doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"extra": true}}},
		{"unencodable", doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"title": make(chan int)}}},
		{"path", "../outside", doc.Hash, Patch{Body: &body}},
		{"instructions", "instructions", doc.Hash, Patch{Body: &body}},
		{"missing document", "missing", doc.Hash, Patch{Body: &body}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := c.Update(tt.id, tt.hash, tt.patch); err == nil {
				t.Fatal("invalid update succeeded")
			}
			data, err := os.ReadFile(filepath.Join(c.Path, doc.filename))
			if err != nil || string(data) != doc.Markdown {
				t.Fatalf("original changed: %q %v", data, err)
			}
			assertNoUpdateArtifacts(t, c.Path)
		})
	}
	// A formatting-only edit also invalidates the hash.
	changed := strings.Replace(doc.Markdown, "title:", "# added comment\ntitle:", 1)
	if err := os.WriteFile(filepath.Join(c.Path, doc.filename), []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Update(doc.ID, doc.Hash, Patch{Body: &body}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if err := os.Symlink(doc.filename, filepath.Join(c.Path, "link.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Update("link", doc.Hash, Patch{Body: &body}); err == nil {
		t.Fatal("symlink accepted")
	}
	assertNoUpdateArtifacts(t, c.Path)
}

func assertNoUpdateArtifacts(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".means-") {
			t.Fatalf("leftover update artifact: %s", entry.Name())
		}
	}
}
