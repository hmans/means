package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCompleteConfiguration(t *testing.T) {
	config := "---\r\ntitle: Memory\r\ndescription: Project memory\r\nschema: {type: object}\r\n---\r\n# Workflow\r\nKeep exact bytes."
	base := t.TempDir()
	source := filepath.Join(base, "template.md")
	if err := os.WriteFile(source, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{source, "-"} {
		t.Run(input, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory")
			var stdout, stderr bytes.Buffer
			args := []string{"init", "--path", path, "--input", input, "--json"}
			if err := Execute(args, strings.NewReader(config), &stdout, &stderr, "test"); err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			filename := filepath.Join(path, "MEANS.md")
			if result["config"] != filename || result["schema"] != nil || stderr.Len() != 0 {
				t.Fatalf("response: %v %s", result, stderr.String())
			}
			stored, err := os.ReadFile(filename)
			if err != nil || string(stored) != config {
				t.Fatal("input changed", err)
			}
			stdout.Reset()
			if err := Execute(args, strings.NewReader(config), &stdout, &stderr, "test"); err == nil {
				t.Fatal("overwrote configuration")
			}
			after, err := os.ReadFile(filename)
			if err != nil || !bytes.Equal(stored, after) {
				t.Fatal("overwrite changed bytes", err)
			}
		})
	}
}

func TestInitConfigurationFailures(t *testing.T) {
	for _, args := range [][]string{
		{"--input", "-"}, {"--input="}, {"--input", "missing-file"},
		{"--input", "-", "--schema", "-"},
	} {
		path := filepath.Join(t.TempDir(), "new")
		var stdout, stderr bytes.Buffer
		command := append([]string{"init", "--path", path, "--json"}, args...)
		if err := Execute(command, strings.NewReader("---\nschema: null\n---\n"), &stdout, &stderr, "test"); err == nil {
			t.Fatalf("accepted %v", args)
		}
		if stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Fatalf("bad error: %s %s", stdout.String(), stderr.String())
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("created path", err)
		}
	}
}
