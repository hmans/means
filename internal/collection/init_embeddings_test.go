package collection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestInitPersistsEmbeddingDefaults(t *testing.T) {
	for _, tt := range []struct {
		config, model string
		dimensions    int
		unchanged     bool
	}{
		{"{provider: openai}", "text-embedding-3-small", 512, false},
		{"{provider: openrouter}", "openai/text-embedding-3-small", 512, false},
		{"{provider: openai, model: text-embedding-3-large}", "text-embedding-3-large", 512, false},
		{"{provider: openai, dimensions: 256}", "text-embedding-3-small", 256, false},
		{"{provider: openai, model: text-embedding-3-large, dimensions: 1024}", "text-embedding-3-large", 1024, true},
		{"false", "", 0, true},
	} {
		t.Run(tt.config, func(t *testing.T) {
			body := "\n# Guidance\r\nKeep exact body bytes."
			input := "---\nschema: {type: integer, const: 9007199254740993}\nembeddings: " + tt.config + "\n---\n" + body
			c, err := InitConfig(t.TempDir(), []byte(input))
			if err != nil {
				t.Fatal(err)
			}
			stored, err := os.ReadFile(filepath.Join(c.Path, ConfigFilename))
			if err != nil {
				t.Fatal(err)
			}
			if tt.unchanged && string(stored) != input {
				t.Fatal("complete/disabled config changed")
			}
			fields, storedBody, err := configFields(stored)
			if err != nil || storedBody != body {
				t.Fatalf("body changed: %q %v", storedBody, err)
			}
			if fields["schema"].(map[string]any)["const"] != json.Number("9007199254740993") {
				t.Fatal("schema number lost precision")
			}
			if tt.model != "" {
				cfg := fields["embeddings"].(map[string]any)
				if cfg["model"] != tt.model || cfg["dimensions"] != json.Number(strconv.Itoa(tt.dimensions)) {
					t.Fatalf("config: %v", cfg)
				}
			}
		})
	}
}
