package userconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEnvironment(t *testing.T) {
	cwd, collection := t.TempDir(), t.TempDir()
	t.Chdir(cwd)
	for _, key := range []string{"MEANS_TEST_SHARED", "MEANS_TEST_FALLBACK", "MEANS_TEST_EXPORTED", "MEANS_TEST_EMPTY"} {
		old, exists := os.LookupEnv(key)
		os.Unsetenv(key)
		t.Cleanup(func() {
			if exists {
				os.Setenv(key, old)
			} else {
				os.Unsetenv(key)
			}
		})
	}
	t.Setenv("MEANS_TEST_EXPORTED", "shell")
	t.Setenv("MEANS_TEST_EMPTY", "")
	os.WriteFile(filepath.Join(collection, ".env"), []byte("export MEANS_TEST_SHARED='collection value'\nMEANS_TEST_EXPORTED=file\nMEANS_TEST_EMPTY=file\n"), 0600)
	os.WriteFile(filepath.Join(cwd, ".env"), []byte("MEANS_TEST_SHARED=cwd\nMEANS_TEST_FALLBACK=\"fallback value\" # comment\n"), 0600)
	if err := LoadEnvironment(collection); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"MEANS_TEST_SHARED": "collection value", "MEANS_TEST_FALLBACK": "fallback value", "MEANS_TEST_EXPORTED": "shell", "MEANS_TEST_EMPTY": ""} {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestLoadEnvironmentErrors(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := LoadEnvironment(dir); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, ".env"), []byte("MEANS_TEST_PENDING=value\nBROKEN='secret"), 0600)
	err := LoadEnvironment(dir)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := os.LookupEnv("MEANS_TEST_PENDING"); exists {
		t.Fatal("partial environment applied")
	}
	os.Remove(filepath.Join(dir, ".env"))
	os.Mkdir(filepath.Join(dir, ".env"), 0700)
	if err := LoadEnvironment(dir); err == nil {
		t.Fatal("directory accepted")
	}
}
