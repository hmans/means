package collection

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeletePreservesOtherFiles(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(map[string]any{"title": "Delete me"}, "Body")
	if err != nil {
		t.Fatal(err)
	}
	child, err := Init(filepath.Join(c.Path, "child"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(c.Path, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(c.Path, "nested", "renamed.md")
	if err := os.Rename(filepath.Join(c.Path, doc.filename), moved); err != nil {
		t.Fatal(err)
	}
	// The same ID in another collection must not block or be affected by deletion.
	files := map[string]string{
		"child/copy.md": doc.Markdown,
		"MEANS.md":      "---\nschema: false\n---\nWorkflow",
		"unrelated.txt": "Keep",
		"broken.md":     "Malformed unrelated document",
		".hidden.md":    doc.Markdown,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(c.Path, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Schema evolution must not prevent deletion of readable documents.
	if err := os.WriteFile(filepath.Join(c.Path, ConfigFilename), []byte(files["MEANS.md"]), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err = Load(c.Path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.Delete(doc.ID, doc.Hash)
	if err != nil || result.ID != doc.ID {
		t.Fatalf("delete: %+v %v", result, err)
	}
	if _, err := os.Stat(moved); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("document remains: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(moved)); err != nil {
		t.Fatal("directory removed", err)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(c.Path, name))
		if err != nil || string(got) != want {
			t.Fatalf("changed %s: %q %v", name, got, err)
		}
	}
	if got, err := os.ReadFile(filepath.Join(c.Path, ConfigFilename)); err != nil || string(got) != files["MEANS.md"] {
		t.Fatalf("schema changed: %s %v", got, err)
	}
	if _, err := child.Read(doc.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Delete(doc.ID, doc.Hash); err == nil {
		t.Fatal("repeated deletion succeeded")
	}
}

func TestDeleteFailuresPreserveDocument(t *testing.T) {
	for _, kind := range []string{"missing hash", "invalid hash", "uppercase hash", "stale hash", "invalid ID", "missing ID", "duplicate", "malformed", "boundary", "hidden", "symlink", "workflow"} {
		t.Run(kind, func(t *testing.T) {
			c, err := Init(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := c.Create(map[string]any{"title": "Keep"}, "Body")
			if err != nil {
				t.Fatal(err)
			}
			id, hash := doc.ID, doc.Hash
			filename := filepath.Join(c.Path, doc.filename)
			want := doc.Markdown
			switch kind {
			case "missing hash":
				hash = ""
			case "invalid hash":
				hash = "bad"
			case "uppercase hash":
				hash = strings.Repeat("A", 64)
			case "stale hash":
				want += "Edited"
				err = os.WriteFile(filename, []byte(want), 0o644)
			case "invalid ID":
				id = "../escape"
			case "missing ID":
				id = "missing"
			case "duplicate":
				err = os.WriteFile(filepath.Join(c.Path, "copy.md"), []byte(want), 0o644)
			case "malformed":
				want = strings.Replace(want, "title: Keep", "title: null", 1)
				err = os.WriteFile(filename, []byte(want), 0o644)
				hash = contentHash(want)
			case "boundary":
				var child *Collection
				child, err = Init(filepath.Join(c.Path, "child"), nil)
				if err == nil {
					moved := filepath.Join(child.Path, doc.filename)
					err = os.Rename(filename, moved)
					filename = moved
				}
			case "workflow":
				moved := filepath.Join(c.Path, ConfigFilename)
				err = os.Rename(filename, moved)
				filename = moved
			case "hidden":
				moved := filepath.Join(c.Path, ".hidden.md")
				err = os.Rename(filename, moved)
				filename = moved
			case "symlink":
				moved := filepath.Join(t.TempDir(), "outside.md")
				err = os.Rename(filename, moved)
				if err == nil {
					err = os.Symlink(moved, filename)
				}
				filename = moved
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = c.Delete(id, hash); err == nil {
				t.Fatal("delete succeeded")
			}
			if kind == "stale hash" && !errors.Is(err, ErrConflict) {
				t.Fatalf("expected conflict: %v", err)
			}
			if kind == "duplicate" && !errors.Is(err, ErrDuplicateID) {
				t.Fatalf("expected duplicate: %v", err)
			}
			got, err := os.ReadFile(filename)
			if err != nil || string(got) != want {
				t.Fatalf("file changed: %q %v", got, err)
			}
		})
	}
}
