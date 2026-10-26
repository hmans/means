package userconfig

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestParse(t *testing.T) {
	for _, input := range []string{"", Example, "{}", "embeddings: false"} {
		cfg, err := Parse([]byte(input))
		if err != nil || cfg.Embeddings != nil {
			t.Fatalf("disabled %q: %+v %v", input, cfg, err)
		}
	}
	for _, tt := range []struct{ provider, model string }{{"openai", "text-embedding-3-small"}, {"openrouter", "openai/text-embedding-3-small"}} {
		cfg, err := Parse([]byte("embeddings: {provider: " + tt.provider + "}"))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Embeddings.Model != tt.model || cfg.Embeddings.Dimensions != 512 {
			t.Fatalf("defaults: %+v", cfg.Embeddings)
		}
	}
	invalid := []string{"!custom {}", "[]", "null", "true", "embeddings: null", "embeddings: true", "embeddings: {}", "unknown: true", "embeddings: false\nembeddings: false", "{}\n---\n{}", "embeddings: {provider: openai, provider: openrouter}", "embeddings: {provider: openai, api_key: secret}", "embeddings: {provider: openai, dimensions: 1.5}", "embeddings: {provider: openai, dimensions: '512'}", "embeddings: {provider: openai, dimensions: 1537}", "embeddings: {provider: openrouter, model: text-embedding-3-small}", "embeddings: {provider: openai, model: openai/text-embedding-3-small}", "embeddings: &e {provider: openai, model: *e}", "embeddings: !custom {provider: openai}"}
	for _, input := range invalid {
		if _, err := Parse([]byte(input)); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestBootstrapNeverOverwrites(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	filename, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Go(func() {
			cfg, warning, err := Open()
			if err != nil || warning != nil || cfg.Embeddings != nil {
				t.Errorf("open: %+v %v %v", cfg, warning, err)
			}
		})
	}
	wg.Wait()
	data, err := os.ReadFile(filename)
	if err != nil || string(data) != Example {
		t.Fatal("incomplete bootstrap")
	}
	info, err := os.Stat(filename)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("config permissions")
	}
	custom := "# keep my exact bytes\nembeddings: {provider: openrouter, dimensions: 256}\n"
	if err := os.WriteFile(filename, []byte(custom), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, warning, err := Open()
	if err != nil || warning != nil || cfg.Embeddings.Dimensions != 256 {
		t.Fatalf("load existing: %+v %v %v", cfg, warning, err)
	}
	after, _ := os.ReadFile(filename)
	if string(after) != custom {
		t.Fatal("overwrote user config")
	}
	if err := os.WriteFile(filename, []byte("embeddings: ["), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Open(); err == nil || !strings.Contains(err.Error(), filename) {
		t.Fatalf("malformed config error: %v", err)
	}
	after, _ = os.ReadFile(filename)
	if string(after) != "embeddings: [" {
		t.Fatal("replaced malformed config")
	}
}

func TestBootstrapFailureIsWarning(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	if err := os.WriteFile(filepath.Join(base, "means"), []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, warning, err := Open()
	if err != nil || warning == nil || cfg.Embeddings != nil {
		t.Fatalf("bootstrap: %+v %v %v", cfg, warning, err)
	}
	if !strings.Contains(warning.Error(), filepath.Join(base, "means", "config.yaml")) {
		t.Fatal("warning lacks path")
	}
}

func TestPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	filename, err := Path()
	if err != nil || filename != filepath.Join(home, ".config", "means", "config.yaml") {
		t.Fatalf("fallback: %s %v", filename, err)
	}
	t.Setenv("XDG_CONFIG_HOME", "relative")
	if _, err := Path(); err == nil {
		t.Fatal("relative config path accepted")
	}
}
