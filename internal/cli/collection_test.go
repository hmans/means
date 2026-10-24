package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/means/internal/collection"
)

func TestInitCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "memory")
	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"init", "--path", dir, "--json"}, nil, &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Path   string `json:"path"`
		Config string `json:"config"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Path != dir || result.Config != filepath.Join(dir, "MEANS.md") || stderr.Len() != 0 {
		t.Fatalf("unexpected initialization response: %s, stderr: %s", stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(result.Config)
	if err != nil {
		t.Fatal(err)
	}
	c, err := collection.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(c.Schema, &schema); err != nil {
		t.Fatal(err)
	}
	if schema["type"] != "object" || !strings.Contains(stdout.String(), "\n  ") {
		t.Fatalf("expected an embedded schema and pretty-printed response: %s, %s", data, stdout.String())
	}
	stdout.Reset()
	if err := Execute([]string{"init", "--path", dir, "--json"}, nil, &stdout, &stderr, "test"); err == nil {
		t.Fatal("reinitialization succeeded")
	}
	if stdout.Len() != 0 || !json.Valid(stderr.Bytes()) || !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("unexpected reinitialization error: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestInitPaths(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"--path", "memory", "init"}, nil, &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Initialized collection") {
		t.Fatalf("unexpected text response: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(cwd, "memory", "MEANS.md")); err != nil {
		t.Fatal(err)
	}
	// Without --path, initialization uses the working directory itself.
	if err := Execute([]string{"init"}, nil, &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cwd, "MEANS.md")); err != nil {
		t.Fatal(err)
	}
}

func TestCollectionCommandErrors(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"schema", "set", "--json"},
		{"init", "--schema", "missing.json", "--json"},
		{"init", "--schema=", "--json"},
		{"init", "unexpected", "--json"},
	} {
		var stdout, stderr bytes.Buffer
		if err := Execute(args, nil, &stdout, &stderr, "test"); err == nil {
			t.Fatalf("Execute(%q) succeeded unexpectedly", args)
		}
		if stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Fatalf("Execute(%q): stdout=%s stderr=%s", args, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat("MEANS.md"); !os.IsNotExist(err) {
		t.Fatalf("failed command wrote a collection: %v", err)
	}
}
