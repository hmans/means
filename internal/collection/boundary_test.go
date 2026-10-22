package collection

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNestedCollectionBoundary(t *testing.T) {
	parent, err := Init(t.TempDir(), []byte(`{"additionalProperties":false}`))
	if err != nil {
		t.Fatal(err)
	}
	child, err := Init(filepath.Join(parent.Path, "ordinary", "child"), []byte(`{"required":["child"]}`))
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(parent.Path, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("MEANS.md", "Parent workflow")
	// Generic schemas must not hide ordinary subdirectories from their owner.
	write("ordinary/schema.json", "unrelated schema")
	write("ordinary/child/MEANS.md", "Child workflow")
	write("ordinary/child/doc.md", "---\nid: same1\ntitle: Child\ndata:\n  child: true\n---\nChild body")
	write("ordinary/local.md", "---\nid: local\ntitle: Local\ndata: {}\n---\nLocal body")
	childDoc, err := child.Read("same1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parent.Read("same1"); err == nil {
		t.Fatal("parent read a child document")
	}
	if _, err := parent.Update("same1", childDoc.Hash, Patch{Frontmatter: map[string]any{"title": "Wrong"}}); err == nil {
		t.Fatal("parent updated a child document")
	}
	root, err := os.OpenRoot(parent.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	// A child ID does not prevent allocation in the parent collection.
	doc, err := createCandidate(root, "same1", map[string]any{"title": "Parent", "data": map[string]any{}}, "Parent body")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parent.Update(doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"title": "Updated"}}); err != nil {
		t.Fatal(err)
	}
	unchanged, err := child.Read("same1")
	if err != nil || unchanged.Hash != childDoc.Hash {
		t.Fatalf("parent update changed child: %+v, %v", unchanged, err)
	}
	for _, tt := range []struct {
		c        *Collection
		count    int
		workflow string
	}{{parent, 2, "Parent workflow"}, {child, 1, "Child workflow"}} {
		listed, err := tt.c.List(0)
		if err != nil || len(listed.Documents) != tt.count || listed.HasMore {
			t.Fatalf("list: %+v, %v", listed, err)
		}
		if len(listed.Diagnostics) != 0 {
			t.Fatalf("unexpected diagnostics: %+v", listed.Diagnostics)
		}
		found, err := tt.c.Search(SearchOptions{})
		if err != nil || len(found.Documents) != tt.count || len(found.Diagnostics) != 0 {
			t.Fatalf("search: %+v, %v", found, err)
		}
		report, err := tt.c.Doctor()
		if err != nil || !report.Healthy || report.Checked != tt.count {
			t.Fatalf("doctor: %+v, %v", report, err)
		}
		info, err := tt.c.Describe()
		if err != nil || info.Instructions != tt.workflow {
			t.Fatalf("guidance: %+v, %v", info, err)
		}
	}
}

func TestBoundaryPresenceAndRemoval(t *testing.T) {
	for _, kind := range []string{"invalid schema", "directory", "dangling symlink"} {
		t.Run(kind, func(t *testing.T) {
			c, err := Init(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(c.Path, "nested")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "broken.md"), []byte("malformed document"), 0o644); err != nil {
				t.Fatal(err)
			}
			check := func(count int) {
				t.Helper()
				report, err := c.Doctor()
				if err != nil || report.Checked != count || report.Healthy != (count == 0) {
					t.Fatalf("doctor: %+v, %v", report, err)
				}
			}
			check(1)
			marker := filepath.Join(dir, SchemaFilename)
			switch kind {
			case "invalid schema":
				err = os.WriteFile(marker, []byte("broken JSON"), 0o644)
			case "directory":
				err = os.Mkdir(marker, 0o755)
			case "dangling symlink":
				err = os.Symlink("missing.json", marker)
			}
			if err != nil {
				t.Fatal(err)
			}
			check(0)
			if _, err := Load(dir); err == nil {
				t.Fatal("invalid child collection loaded")
			}
			if err := os.Remove(marker); err != nil {
				t.Fatal(err)
			}
			check(1)
		})
	}
}
