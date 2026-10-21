package collection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectionDescription(t *testing.T) {
	for _, schema := range []string{`true`, `false`, `{"type":"object"}`, `{"title":"  ","properties":{"title":{"title":"Document title"}}}`} {
		t.Run(schema, func(t *testing.T) {
			c, err := Init(t.TempDir(), []byte(schema))
			if err != nil {
				t.Fatal(err)
			}
			info, err := c.Describe()
			if err != nil {
				t.Fatal(err)
			}
			if info.Title != filepath.Base(c.Path) || info.Description != "" || info.Instructions != "" {
				t.Fatalf("unexpected fallback: %+v", info)
			}
		})
	}
	c, err := Init(t.TempDir(), []byte(`{"title":" Project issues ","description":" Planned work. ","type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, guidance := range []string{"# Workflow\n\nCheck duplicates first.\n", "Updated instructions.\n", ""} {
		if err := os.WriteFile(filepath.Join(c.Path, InstructionsFilename), []byte(guidance), 0o644); err != nil {
			t.Fatal(err)
		}
		info, err := c.Describe()
		if err != nil {
			t.Fatal(err)
		}
		if info.Title != "Project issues" || info.Description != "Planned work." || info.Instructions != guidance {
			t.Fatalf("description = %+v", info)
		}
		list, err := c.List(0)
		if err != nil || len(list.Documents) != 0 || list.HasMore {
			t.Fatalf("instructions listed: %+v %v", list, err)
		}
		if _, err := c.Read("instructions"); err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("read instructions: %v", err)
		}
	}
}

func TestWorkflowFilenameScope(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(c.Path, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, id := range map[string]string{
		"MEANS.md":        "root1",
		"instructions.md": "old01",
		"nested/MEANS.md": "nest1",
	} {
		content := "---\nid: " + id + "\ntitle: Example\ndata: {}\n---\nWorkflow\n"
		if err := os.WriteFile(filepath.Join(c.Path, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.Read("root1"); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("root workflow must not be readable as a document: %v", err)
	}
	for _, id := range []string{"old01", "nest1"} {
		if _, err := c.Read(id); err != nil {
			t.Fatalf("read ordinary document %s: %v", id, err)
		}
	}
	if err := os.Remove(filepath.Join(c.Path, "MEANS.md")); err != nil {
		t.Fatal(err)
	}
	info, err := c.Describe()
	if err != nil || info.Instructions != "" {
		t.Fatalf("legacy filename must not supply guidance: %+v, %v", info, err)
	}
}

func TestInvalidCollectionInstructions(t *testing.T) {
	for _, kind := range []string{"directory", "symlink", "invalid UTF-8"} {
		t.Run(kind, func(t *testing.T) {
			c, err := Init(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(c.Path, InstructionsFilename)
			switch kind {
			case "directory":
				err = os.Mkdir(path, 0o755)
			case "symlink":
				err = os.Symlink(filepath.Join(c.Path, SchemaFilename), path)
			case "invalid UTF-8":
				err = os.WriteFile(path, []byte{0xff}, 0o644)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Describe(); err == nil || !strings.Contains(err.Error(), InstructionsFilename) {
				t.Fatalf("invalid instructions accepted: %v", err)
			}
		})
	}
}
