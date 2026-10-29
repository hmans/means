package collection

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAppConfigValidationBeforeWrites(t *testing.T) {
	for _, app := range []string{
		"null", "[]", "true", "{unknown: true}", "{views: null}",
		"{views: [null]}", "{views: [{title: 42}]}",
		"{views: [{title: X, filter: null}]}", "{views: [{title: X, columns: title}]}",
		"{views: [{title: X, columns: [null]}]}", "{views: [{title: X, columns: [body]}]}",
		"{views: [{title: X, unknown: true}]}", "{views: [{title: X}, {title: X}]}",
	} {
		path := filepath.Join(t.TempDir(), "new")
		if _, err := InitConfig(path, []byte("---\nschema: true\napp: "+app+"\n---\n")); err == nil {
			t.Errorf("accepted app %s", app)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("invalid app created collection")
		}
	}
}

func TestAppConfigSnapshotAndSchemaReplacement(t *testing.T) {
	content := []byte("---\ntitle: Example\nschema: true\napp:\n  views:\n    - title: Open\n      filter: 'invalid expression ('\n      columns: [title, data.status]\n---\nExact workflow body.\n")
	c, err := InitConfig(t.TempDir(), content)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(c.Path, ConfigFilename))
	if err != nil || !bytes.Equal(content, stored) {
		t.Fatal("init changed complete configuration bytes")
	}
	if len(c.App.Views) != 1 || c.App.Views[0].Title != "Open" {
		t.Fatal("app not loaded")
	}
	if _, err := c.Search(SearchOptions{}); err != nil {
		t.Fatal("expression syntax blocked collection operations", err)
	}
	if _, err := SetSchema(c.Path, []byte(`{"type":"object"}`), false); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(c.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.App, reloaded.App) || reloaded.Describe().Instructions != "Exact workflow body.\n" {
		t.Fatal("schema replacement changed app or workflow")
	}
	if err := os.WriteFile(filepath.Join(c.Path, ConfigFilename), []byte("---\nschema: true\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
	reloaded, err = Load(c.Path)
	if err != nil || len(reloaded.App.Views) != 0 || len(c.App.Views) != 1 {
		t.Fatal("optional app or snapshot semantics failed")
	}
}
