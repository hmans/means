package collection

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitAndLoad(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "memory")
	c, err := Init(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Path != dir || loaded.Path != dir {
		t.Fatalf("collection paths = %q, %q; want %q", c.Path, loaded.Path, dir)
	}
	data, err := os.ReadFile(filepath.Join(dir, SchemaFilename))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if schema["type"] != "object" || schema["schema"] != nil {
		t.Fatalf("expected a standalone JSON Schema, got %s", data)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != SchemaFilename {
		t.Fatalf("expected only means.schema.json, got %v, %v", entries, err)
	}
	for _, tt := range []struct {
		name  string
		value map[string]any
		valid bool
	}{
		{"empty data", map[string]any{}, true},
		{"tags", map[string]any{"tags": []any{"project", "decision"}}, true},
		{"wrong tags type", map[string]any{"tags": "project"}, false},
		{"wrong tag item", map[string]any{"tags": []any{true}}, false},
		{"unknown field", map[string]any{"typo": true}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := loaded.ValidateData(tt.value); (err == nil) != tt.valid {
				t.Fatalf("Validate() = %v, want valid=%v", err, tt.valid)
			}
		})
	}
}

func TestInitDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, SchemaFilename)
	original := []byte("existing schema, even if invalid\n")
	if err := os.WriteFile(filename, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(dir, nil); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Init() = %v, want existing schema error", err)
	}
	after, err := os.ReadFile(filename)
	if err != nil || !bytes.Equal(after, original) {
		t.Fatalf("existing schema changed: %q, %v", after, err)
	}
}

func TestCustomSchemaAndReload(t *testing.T) {
	dir := t.TempDir()
	schema := []byte(`{
		"type": "object",
		"$defs": {"status": {"enum": ["open", "closed"]}},
		"properties": {"status": {"$ref": "#/$defs/status"}},
		"required": ["status"]
	}`)
	if err := os.WriteFile(filepath.Join(dir, SchemaFilename), schema, 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateData(map[string]any{"status": "open"}); err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateData(map[string]any{"status": "unknown"}); err == nil {
		t.Fatal("invalid enum value was accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, SchemaFilename), []byte(`{"type":"object","required":["summary"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err = Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateData(map[string]any{"status": "open"}); err == nil {
		t.Fatal("Load did not pick up the edited schema")
	}
	if err := c.ValidateData(map[string]any{"summary": "Updated"}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadInvalidSchema(t *testing.T) {
	for _, schema := range []string{
		``, `{`, `null`, `[]`, `{"type":"nonsense"}`,
		`{"required":"title"}`, `{"properties":{"title":{"type":42}}}`,
		`{} {}`, `{"$ref":"#/$defs/missing"}`,
		`{"$ref":"other.json"}`, `{"$ref":"https://example.com/schema.json"}`,
	} {
		t.Run(schema, func(t *testing.T) {
			dir := t.TempDir()
			filename := filepath.Join(dir, SchemaFilename)
			if err := os.WriteFile(filename, []byte(schema), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), SchemaFilename) {
				t.Fatalf("Load() = %v, want an error identifying means.schema.json", err)
			}
			after, err := os.ReadFile(filename)
			if err != nil || string(after) != schema {
				t.Fatalf("loading invalid schema modified it: %q, %v", after, err)
			}
		})
	}
}

func TestLoadDoesNotSearchParents(t *testing.T) {
	dir := t.TempDir()
	if _, err := Init(dir, nil); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(child); err == nil || !strings.Contains(err.Error(), "means init") {
		t.Fatalf("Load() = %v, want missing collection diagnostic", err)
	}
}
