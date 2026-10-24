package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentsGuideWithoutCollection(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	var text, structured, stderr bytes.Buffer
	if err := Execute([]string{"agents"}, nil, &text, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if err := Execute([]string{"agents", "--json", "--path", filepath.Join(dir, "missing")}, nil, &structured, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Guide string `json:"guide"`
	}
	if err := json.Unmarshal(structured.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Guide != text.String() || result.Guide == "" || stderr.Len() != 0 {
		t.Fatal("guide output mismatch")
	}
	for _, topic := range []string{"MEANS.md", "--input", "project instructions", "skill", "means mcp", "info", "doctor", "hash", "--with-body"} {
		if !strings.Contains(result.Guide, topic) {
			t.Errorf("missing topic %q", topic)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("guide wrote files: %v %v", entries, err)
	}
}

func TestAgentGuideDiscovery(t *testing.T) {
	for _, command := range [][]string{{"--help"}, {"init", "--help"}, {"agents", "--help"}} {
		var out, stderr bytes.Buffer
		if err := Execute(append(command, "--json"), nil, &out, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		if !json.Valid(out.Bytes()) || !strings.Contains(out.String(), "means agents") {
			t.Fatalf("missing discovery: %s", &out)
		}
	}
	for _, jsonOutput := range []bool{false, true} {
		var out, stderr bytes.Buffer
		args := []string{"init", "--path", filepath.Join(t.TempDir(), "memory")}
		if jsonOutput {
			args = append(args, "--json")
		}
		if err := Execute(args, nil, &out, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		if jsonOutput {
			var result struct {
				NextSteps []string `json:"next_steps"`
			}
			if err := json.Unmarshal(out.Bytes(), &result); err != nil || len(result.NextSteps) != 1 || result.NextSteps[0] != "means agents" {
				t.Fatalf("next steps: %s %v", &out, err)
			}
		} else if !strings.Contains(out.String(), "means agents") {
			t.Fatal("missing next step")
		}
	}
}
