package userconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unsetenv removes variables for the test and restores them afterwards.
func unsetenv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func TestLoadEnvironment(t *testing.T) {
	cwd, collection := t.TempDir(), t.TempDir()
	t.Chdir(cwd)
	ignored := []string{"VISUAL", "EDITOR", "MEANS_EMBEDDING_BASE_URL", "XDG_CONFIG_HOME", "NO_COLOR", "MEANS_TEST_OTHER"}
	unsetenv(t, append([]string{"OPENAI_API_KEY", "OPENROUTER_API_KEY", "MEANS_EMBEDDING_API_KEY"}, ignored...)...)
	os.WriteFile(filepath.Join(collection, ".env"), []byte("export OPENAI_API_KEY='collection value'\nVISUAL=/tmp/evil\nEDITOR=/tmp/evil\nMEANS_EMBEDDING_BASE_URL=https://evil.example\nXDG_CONFIG_HOME=/tmp/evil\n"), 0600)
	os.WriteFile(filepath.Join(cwd, ".env"), []byte("OPENAI_API_KEY=cwd\nOPENROUTER_API_KEY=\"fallback value\" # comment\nMEANS_EMBEDDING_API_KEY=override\nNO_COLOR=1\nMEANS_TEST_OTHER=value\n"), 0600)
	if err := LoadEnvironment(collection); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"OPENAI_API_KEY": "collection value", "OPENROUTER_API_KEY": "fallback value", "MEANS_EMBEDDING_API_KEY": "override"} {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	for _, key := range ignored {
		if value, exists := os.LookupEnv(key); exists {
			t.Errorf("%s = %q loaded from .env", key, value)
		}
	}
}

func TestLoadEnvironmentPrecedence(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("OPENAI_API_KEY", "shell")
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("VISUAL", "shell-editor")
	t.Setenv("MEANS_EMBEDDING_BASE_URL", "https://shell.example")
	unsetenv(t, "MEANS_EMBEDDING_API_KEY")
	os.WriteFile(filepath.Join(dir, ".env"), []byte("OPENAI_API_KEY=file\nOPENROUTER_API_KEY=file\nVISUAL=file\nMEANS_EMBEDDING_BASE_URL=https://file.example\n"), 0600)
	if err := LoadEnvironment(dir); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"OPENAI_API_KEY": "shell", "OPENROUTER_API_KEY": "", "VISUAL": "shell-editor", "MEANS_EMBEDDING_BASE_URL": "https://shell.example"} {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestLoadEnvironmentErrors(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	unsetenv(t, "OPENAI_API_KEY")
	if err := LoadEnvironment(dir); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, ".env"), []byte("OPENAI_API_KEY=value\nBROKEN='secret"), 0600)
	err := LoadEnvironment(dir)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := os.LookupEnv("OPENAI_API_KEY"); exists {
		t.Fatal("partial environment applied")
	}
	os.Remove(filepath.Join(dir, ".env"))
	os.Mkdir(filepath.Join(dir, ".env"), 0700)
	if err := LoadEnvironment(dir); err == nil {
		t.Fatal("directory accepted")
	}
}
