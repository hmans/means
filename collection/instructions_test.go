package collection

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetInstructionsPreservesConfiguration(t *testing.T) {
	for i, prefix := range []string{
		"---\n# Keep this comment\ntitle: 'Notes'\ndescription: Notes and decisions\nschema: {properties: {n: {const: 9007199254740993}}}\napp: {views: [{title: All, columns: [title]}]}\n---\n",
		"---\r\nschema: true\r\n---\r\n",
		"---\nschema: false\n---",
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c, err := InitConfig(t.TempDir(), []byte(prefix))
			if err != nil {
				t.Fatal(err)
			}
			filename := filepath.Join(c.Path(), ConfigFilename)
			if err := os.Chmod(filename, 0o640); err != nil {
				t.Fatal(err)
			}
			// Invalid documents must not prevent workflow edits.
			if err := os.WriteFile(filepath.Join(c.Path(), "broken.md"), []byte("bad document"), 0o644); err != nil {
				t.Fatal(err)
			}
			for _, body := range []string{"\n# Workflow\r\n\nCheck existing notes. 📝\n---\n", "", "No final newline"} {
				loaded, err := Load(c.Path())
				if err != nil {
					t.Fatal(err)
				}
				before, err := os.ReadFile(filename)
				if err != nil || loaded.Describe().Hash != contentHash(string(before)) {
					t.Fatal("info hash does not cover stored bytes", err)
				}
				result, err := loaded.SetInstructions(context.Background(), loaded.Describe().Hash, body)
				if err != nil {
					t.Fatal(err)
				}
				if body != "" && !strings.HasSuffix(prefix, "\n") {
					prefix += "\n"
				}
				stored, err := os.ReadFile(filename)
				if err != nil || string(stored) != prefix+body || result.Hash != contentHash(string(stored)) || result.Instructions != body {
					t.Fatalf("stored %q, result %+v, err %v", stored, result, err)
				}
				stat, err := os.Stat(filename)
				if err != nil || stat.Mode().Perm() != 0o640 {
					t.Fatal("permissions changed", err)
				}
			}
		})
	}
}

func TestSetInstructionsRejectsStaleAndInvalidWrites(t *testing.T) {
	for _, change := range []string{"body", "schema", "title", "symlink", "directory", "missing"} {
		t.Run(change, func(t *testing.T) {
			c, err := Init(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			filename := filepath.Join(c.Path(), ConfigFilename)
			original, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			for _, hash := range []string{"", "bad", strings.Repeat("0", 64)} {
				if _, err := c.SetInstructions(context.Background(), hash, "bad"); err == nil {
					t.Fatal("invalid or stale hash accepted")
				}
			}
			if _, err := c.SetInstructions(context.Background(), c.Describe().Hash, string([]byte{0xff})); err == nil {
				t.Fatal("invalid UTF-8 accepted")
			}
			after, err := os.ReadFile(filename)
			if err != nil || string(after) != string(original) {
				t.Fatal("failed write changed file", err)
			}
			switch change {
			case "body", "schema", "title":
				content := "---\nschema: true\n---\nNew workflow"
				if change == "schema" {
					content = "---\nschema: false\n---\n" + c.body
				} else if change == "title" {
					content = "---\ntitle: New title\nschema: true\n---\n" + c.body
				}
				if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
				if _, err := c.SetInstructions(context.Background(), c.Describe().Hash, "Replacement"); !errors.Is(err, ErrConflict) {
					t.Fatalf("expected conflict, got %v", err)
				}
				stored, err := os.ReadFile(filename)
				if err != nil || string(stored) != content {
					t.Fatal("conflict changed file", err)
				}
				return
			case "symlink", "directory", "missing":
				if err := os.Remove(filename); err != nil {
					t.Fatal(err)
				}
				if change == "symlink" {
					target := filepath.Join(t.TempDir(), "target.md")
					if err := os.WriteFile(target, original, 0o644); err != nil {
						t.Fatal(err)
					}
					err = os.Symlink(target, filename)
				} else if change == "directory" {
					err = os.Mkdir(filename, 0o755)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := c.SetInstructions(context.Background(), c.Describe().Hash, "Replacement"); err == nil {
				t.Fatal("invalid file accepted")
			}
		})
	}
}
