package collection

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigValidationBeforeWrites(t *testing.T) {
	for _, header := range []string{
		"", "# comment", "[]", "schema: null", "schema: []", "schema: string",
		"schema: true\ntitle: 42", "schema: true\ndescription: null",
		"schema: true\nunknown: value", "schema: true\nschema: false",
		"schema: {type: invalid}", "schema: {$ref: 'other.json'}",
		"schema: {$ref: '#/$defs/missing'}", "schema: [",
		"schema: &schema {}\ntitle: *schema", "schema: {minimum: .inf}",
	} {
		t.Run(header, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "new")
			if _, err := InitConfig(path, []byte("---\n"+header+"\n---\n")); err == nil {
				t.Fatal("accepted invalid configuration")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("created path: %v", err)
			}
		})
	}
}

func TestConfigIdentityAndSnapshot(t *testing.T) {
	for _, schema := range []string{"true", "false", "{title: Schema title, description: Schema description}"} {
		t.Run(schema, func(t *testing.T) {
			dir := t.TempDir()
			original := []byte("---\ntitle: '  '\nschema: " + schema + "\n---\n")
			c, err := InitConfig(dir, original)
			if err != nil {
				t.Fatal(err)
			}
			info := c.Describe()
			if info.Title != filepath.Base(dir) || info.Description != "" || info.Instructions != "" {
				t.Fatalf("identity: %+v", info)
			}
			stored, err := os.ReadFile(filepath.Join(dir, ConfigFilename))
			if err != nil || !bytes.Equal(stored, original) {
				t.Fatal("input bytes changed", err)
			}
			updated := []byte("---\ntitle: Updated\ndescription: New summary\nschema: false\n---\nNew guidance\n")
			if err := os.WriteFile(filepath.Join(dir, ConfigFilename), updated, 0o644); err != nil {
				t.Fatal(err)
			}
			snapshot := c.Describe()
			if *snapshot != *info {
				t.Fatal("loaded snapshot changed")
			}
			loaded, err := Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			current := loaded.Describe()
			if current.Title != "Updated" || current.Description != "New summary" || current.Instructions != "New guidance\n" || string(loaded.Schema) != "false" {
				t.Fatalf("reload: %+v %s", current, loaded.Schema)
			}
		})
	}
}

func TestSchemaReplacementPreservesConfig(t *testing.T) {
	body := "\r\n# Guidance\r\nKeep spacing.  \n---\nNo final newline"
	original := "---\r\ntitle: ' Collection '\r\ndescription: ' Summary '\r\nschema: {type: invalid}\r\n---\r\n" + body
	dir := t.TempDir()
	filename := filepath.Join(dir, ConfigFilename)
	if err := os.WriteFile(filename, []byte(original), 0o640); err != nil {
		t.Fatal(err)
	}
	proposal := []byte(`{"$defs":{"n":{"const":9007199254740993}},"properties":{"n":{"$ref":"#/$defs/n"},"ratio":{"const":0.1234567890123456789}}}`)
	if _, err := SetSchema(dir, proposal, false); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.config["title"] != " Collection " || c.config["description"] != " Summary " || c.body != body {
		t.Fatalf("configuration changed: %+v %q", c.config, c.body)
	}
	if err := c.ValidateData(map[string]any{"n": json.Number("9007199254740993"), "ratio": json.Number("0.1234567890123456789")}); err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateData(map[string]any{"n": json.Number("9007199254740992")}); err == nil {
		t.Fatal("rounded constraint")
	}
	stat, err := os.Stat(filename)
	if err != nil || stat.Mode().Perm() != 0o640 {
		t.Fatal("permissions changed", err)
	}
	stored, err := os.ReadFile(filename)
	if err != nil || !strings.Contains(string(stored), "9007199254740993") || !strings.Contains(string(stored), "0.1234567890123456789") {
		t.Fatal("precision lost", err)
	}
}

func TestLegacySchemaDoesNotLoad(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "means.schema.json"), []byte(`true`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("loaded legacy schema")
	}
}

func TestConfigReplacementRejectsConcurrentEdits(t *testing.T) {
	for _, change := range []string{"title", "description", "body", "replacement"} {
		t.Run(change, func(t *testing.T) {
			dir := t.TempDir()
			original := "---\ntitle: Original\ndescription: Summary\nschema: true\n---\nBody"
			if _, err := InitConfig(dir, []byte(original)); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			stat, err := root.Lstat(ConfigFilename)
			if err != nil {
				t.Fatal(err)
			}
			updated := original
			switch change {
			case "title":
				updated = strings.Replace(original, "Original", "Edited", 1)
			case "description":
				updated = strings.Replace(original, "Summary", "Edited", 1)
			case "body":
				updated += " edited"
			case "replacement":
				// Keep the old inode alive so its number cannot be reused.
				if err := root.Rename(ConfigFilename, "old-config"); err != nil {
					t.Fatal(err)
				}
			}
			if err := root.WriteFile(ConfigFilename, []byte(updated), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := replaceConfig(root, dir, stat, []byte(original), "---\nschema: false\n---\n"); !errors.Is(err, ErrConflict) {
				t.Fatalf("expected conflict: %v", err)
			}
			stored, err := root.ReadFile(ConfigFilename)
			if err != nil || string(stored) != updated {
				t.Fatal("concurrent edit lost", err)
			}
			matches, err := filepath.Glob(filepath.Join(dir, ".means-schema-*.tmp"))
			if err != nil || len(matches) != 0 {
				t.Fatalf("temporary files remain: %v %v", matches, err)
			}
		})
	}
}

func TestLoadMissingConfigSuggestsInit(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "no collection configuration") || !strings.Contains(err.Error(), "run means init --path") {
		t.Fatalf("unhelpful error: %v", err)
	}
}

func TestConfigEncodingOrdersIdentityFirst(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "memory")
	if _, err := Init(dir, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := SetSchema(dir, []byte(`{"type":"object"}`), false); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(dir, ConfigFilename)
	stored, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(stored), "---\n", "---\ndescription: Summary\n", 1)
	if err := os.WriteFile(filename, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SetSchema(dir, []byte(`true`), false); err != nil {
		t.Fatal(err)
	}
	stored, err = os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	want := "---\ntitle: memory\ndescription: Summary\nschema: true\n---\n"
	if !strings.HasPrefix(string(stored), want) {
		t.Fatalf("field order: %q", stored)
	}
}

func TestSchemaInputSurvivesYAMLEncoding(t *testing.T) {
	schema := "{\n\t\"type\": \"object\",\n\t\"properties\": {\n" +
		"\t\t\"html\": {\"const\": \"<a href=\\\"x\\\">&amp;</a>\"},\n" +
		"\t\t\"escaped\": {\"const\": \"\\u00e9\\/\\u2028\\t\"},\n" +
		"\t\t\"emoji\": {\"const\": \"🌱\"},\n" +
		"\t\t\"surrogate\": {\"const\": \"\\ud83c\\udf31\"},\n" +
		"\t\t\"<<\": {\"properties\": {\"<<\": {\"enum\": [\"<<\", \"yes\", \"null\", \"123\", \"#/$defs/x\", \"- item\"]}}}\n" +
		"\t}\n}\n"
	valid := map[string]any{"html": `<a href="x">&amp;</a>`, "escaped": "é/ \t", "emoji": "🌱", "surrogate": "🌱", "<<": map[string]any{"<<": "<<"}}
	for _, install := range []string{"init", "schema set"} {
		t.Run(install, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "c")
			var err error
			if install == "init" {
				_, err = Init(dir, []byte(schema))
			} else if _, err = Init(dir, nil); err == nil {
				_, err = SetSchema(dir, []byte(schema), false)
			}
			if err != nil {
				t.Fatal(err)
			}
			c, err := Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			if err := c.ValidateData(valid); err != nil {
				t.Fatal(err)
			}
			for _, value := range []any{"yes", "null", "123", "#/$defs/x", "- item"} {
				if err := c.ValidateData(map[string]any{"<<": map[string]any{"<<": value}}); err != nil {
					t.Fatalf("enum %q: %v", value, err)
				}
			}
			for key, value := range map[string]any{"html": "<a>", "escaped": "é/", "<<": map[string]any{"<<": "<"}} {
				if err := c.ValidateData(map[string]any{key: value}); err == nil {
					t.Fatalf("%s: accepted %v", key, value)
				}
			}
		})
	}
}

func TestSchemaWithDuplicateKeysIsRejected(t *testing.T) {
	schema := []byte(`{"type":"object","type":"string"}`)
	path := filepath.Join(t.TempDir(), "new")
	if _, err := Init(path, schema); err == nil {
		t.Fatal("init accepted duplicate keys")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("created path: %v", err)
	}
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(c.Path, ConfigFilename)
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SetSchema(c.Path, schema, true); err == nil {
		t.Fatal("schema set accepted duplicate keys")
	}
	stored, err := os.ReadFile(filename)
	if err != nil || !bytes.Equal(stored, original) {
		t.Fatal("rejection changed configuration", err)
	}
	assertNoUpdateArtifacts(t, c.Path)
}

func TestConfigAcceptsJSONSchemaInYAML(t *testing.T) {
	schema := `{"type": "object", "properties": {"n": {"$ref": "#/$defs/n"}}, "$defs": {"n": {"const": 9007199254740993}}}`
	for name, frontmatter := range map[string]string{
		"single line":   "schema: " + schema,
		"multiline":     "schema: {\n  \"type\": \"object\",\n  \"properties\": {\"n\": {\"$ref\": \"#/$defs/n\"}},\n  \"$defs\": {\"n\": {\"const\": 9007199254740993}}\n}",
		"indented":      "schema:\n  " + schema,
		"beside fields": "title: JSON\ndescription: \"Summary\"\nschema: " + schema,
	} {
		t.Run(name, func(t *testing.T) {
			c, err := InitConfig(t.TempDir(), []byte("---\n"+frontmatter+"\n---\n"))
			if err != nil {
				t.Fatal(err)
			}
			if err := c.ValidateData(map[string]any{"n": json.Number("9007199254740993")}); err != nil {
				t.Fatal(err)
			}
			if err := c.ValidateData(map[string]any{"n": json.Number("9007199254740992")}); err == nil {
				t.Fatal("reference or precision lost")
			}
		})
	}
}
