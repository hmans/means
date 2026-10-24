package collection

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetSchemaCompatibility(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(map[string]any{"title": "Keep"}, "Body")
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(c.Path, ConfigFilename)
	if err := os.Chmod(filename, 0o640); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	proposal := []byte("{\n  \"required\": [\"status\"]\n}\n")
	result, err := SetSchema(c.Path, proposal, false)
	if !errors.Is(err, ErrSchemaIncompatible) || result.Installed || result.Healthy || result.Checked != 1 || len(result.Diagnostics) != 1 || result.Diagnostics[0].ID != doc.ID || result.Diagnostics[0].Path != doc.filename {
		t.Fatalf("rejection: %+v %v", result, err)
	}
	stored, err := os.ReadFile(filename)
	if err != nil || string(stored) != string(original) {
		t.Fatal("rejection changed schema", err)
	}
	result, err = SetSchema(c.Path, proposal, true)
	if err != nil || !result.Installed || result.Healthy || len(result.Diagnostics) != 1 {
		t.Fatalf("force: %+v %v", result, err)
	}
	stored, err = os.ReadFile(filename)
	if err != nil || !strings.Contains(string(stored), "required:") {
		t.Fatal("schema bytes changed", err)
	}
	stat, err := os.Stat(filename)
	if err != nil || stat.Mode().Perm() != 0o640 {
		t.Fatalf("permissions: %v %v", stat, err)
	}
	current, err := Load(c.Path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := current.Read(doc.ID)
	if err != nil || read.Hash != doc.Hash {
		t.Fatal("document changed", err)
	}
	if _, err := current.Update(doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"data": map[string]any{"status": "open"}}}); err != nil {
		t.Fatal(err)
	}
	result, err = SetSchema(c.Path, proposal, false)
	if err != nil || !result.Installed || !result.Healthy {
		t.Fatalf("migrated: %+v %v", result, err)
	}
	assertNoUpdateArtifacts(t, c.Path)
}

func TestSchemaKeepsSuppliedKeyOrder(t *testing.T) {
	want := "schema:\n  type: object\n  properties:\n    z:\n      maximum: 12345678901234567890.5\n    a: {}\n  required:\n    - z\n"
	schema := []byte(`{"type":"object","properties":{"z":{"maximum":12345678901234567890.5},"a":{}},"required":["z"]}`)
	c, err := Init(t.TempDir(), schema)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(c.Path, ConfigFilename)
	stored, err := os.ReadFile(filename)
	if err != nil || !strings.Contains(string(stored), want) {
		t.Fatalf("init order:\n%s %v", stored, err)
	}
	if _, err := SetSchema(c.Path, []byte(strings.Replace(string(schema), `"z"]`, `"a","z"]`, 1)), false); err != nil {
		t.Fatal(err)
	}
	stored, err = os.ReadFile(filename)
	if err != nil || !strings.Contains(string(stored), strings.Replace(want, "    - z\n", "    - a\n    - z\n", 1)) {
		t.Fatalf("schema set order:\n%s %v", stored, err)
	}
}

func TestSetSchemaQuotesMergeKeyProperty(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SetSchema(c.Path, []byte(`{"properties":{"<<":{"const":"<<"}}}`), false); err != nil {
		t.Fatal(err)
	}
	current, err := Load(c.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := current.ValidateData(map[string]any{"<<": "<<"}); err != nil {
		t.Fatal(err)
	}
	if err := current.ValidateData(map[string]any{"<<": "other"}); err == nil {
		t.Fatal("schema lost its << property")
	}
}

func TestSetSchemaInvalidAndMissing(t *testing.T) {
	for _, proposal := range []string{"", "{", "null", `{"type":42}`, `{"$ref":"missing.json"}`} {
		for _, force := range []bool{false, true} {
			c, err := Init(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(filepath.Join(c.Path, ConfigFilename))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := SetSchema(c.Path, []byte(proposal), force); err == nil {
				t.Fatalf("accepted %q", proposal)
			}
			stored, err := os.ReadFile(filepath.Join(c.Path, ConfigFilename))
			if err != nil || string(stored) != string(original) {
				t.Fatal("invalid proposal changed schema", err)
			}
			assertNoUpdateArtifacts(t, c.Path)
		}
	}
	path := filepath.Join(t.TempDir(), "missing")
	if _, err := SetSchema(path, []byte(`true`), true); err == nil {
		t.Fatal("created missing collection")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("created directory: %v", err)
	}
	if _, err := SetSchema(t.TempDir(), []byte(`true`), true); err == nil {
		t.Fatal("created missing schema")
	}
}

func TestSetSchemaBoundariesAndDiagnostics(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	child, err := Init(filepath.Join(c.Path, "child"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := child.Create(map[string]any{"title": "Child"}, ""); err != nil {
		t.Fatal(err)
	}
	// Empty parent accepts false; child documents and workflow are excluded.
	if err := os.WriteFile(filepath.Join(c.Path, ConfigFilename), []byte("---\nschema: true\n---\nWorkflow"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := SetSchema(c.Path, []byte(`false`), false)
	if err != nil || !result.Healthy || result.Checked != 0 {
		t.Fatalf("child leaked: %+v %v", result, err)
	}
	for name, body := range map[string]string{"a.md": "---\nid: same1\ntitle: A\ndata: {}\n---\n", "b.md": "---\nid: same1\ntitle: B\ndata: {}\n---\n", "broken.md": "broken"} {
		if err := os.WriteFile(filepath.Join(c.Path, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, force := range []bool{false, true} {
		result, err := SetSchema(c.Path, []byte(`true`), force)
		if result == nil || result.Installed != force || result.Checked != 3 || len(result.Diagnostics) != 3 {
			t.Fatalf("diagnostics: %+v %v", result, err)
		}
		if force && err != nil || !force && !errors.Is(err, ErrSchemaIncompatible) {
			t.Fatal(err)
		}
	}
}

func TestSetSchemaRepairsInvalidSchemaAndRejectsSpecialFiles(t *testing.T) {
	for _, kind := range []string{"invalid", "missing schema", "malformed YAML", "invalid title", "symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			c, err := Init(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			filename := filepath.Join(c.Path, ConfigFilename)
			if err := os.Remove(filename); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "invalid":
				err = os.WriteFile(filename, []byte("---\nschema: {type: nonsense}\n---\n"), 0o644)
			case "missing schema":
				err = os.WriteFile(filename, []byte("---\ntitle: Repair\n---\n"), 0o644)
			case "malformed YAML":
				err = os.WriteFile(filename, []byte("---\nschema: [\n---\n"), 0o644)
			case "invalid title":
				err = os.WriteFile(filename, []byte("---\ntitle: 42\nschema: true\n---\n"), 0o644)
			case "directory":
				err = os.Mkdir(filename, 0o755)
			case "symlink":
				err = os.Symlink("missing", filename)
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = SetSchema(c.Path, []byte(`true`), true)
			if (err == nil) != (kind == "invalid" || kind == "missing schema") {
				t.Fatalf("%s: %v", kind, err)
			}
		})
	}
}
