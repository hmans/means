package collection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectionRejectsLegacyEmbeddings(t *testing.T) {
	for _, value := range []string{"false", "{provider: openrouter}", "{provider: openai, model: text-embedding-3-small, dimensions: 512}"} {
		dir := t.TempDir()
		input := []byte("---\nschema: true\nembeddings: " + value + "\n---\nGuidance\n")
		if _, err := InitConfig(dir, input); err == nil || !strings.Contains(err.Error(), "user config") {
			t.Fatalf("missing migration error: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, ConfigFilename)); !os.IsNotExist(err) {
			t.Fatal("invalid input created file")
		}
		if err := os.WriteFile(filepath.Join(dir, ConfigFilename), input, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "remove it from MEANS.md") {
			t.Fatalf("load migration error: %v", err)
		}
	}
}

func TestInitPreservesCompleteConfigBytes(t *testing.T) {
	input := []byte("---\n# comment\nschema: {type: integer, const: 9007199254740993}\n---\n\n# Guidance\r\nExact body.")
	c, err := InitConfig(t.TempDir(), input)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(c.Path, ConfigFilename))
	if err != nil || string(stored) != string(input) {
		t.Fatal("init changed complete input")
	}
}
