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
	if err := Execute([]string{"help", "agents"}, nil, &text, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if err := Execute([]string{"help", "agents", "--json", "--path", filepath.Join(dir, "missing")}, nil, &structured, &stderr, "test"); err != nil {
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

func TestHelpAudiences(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		run := func(args ...string) string {
			t.Helper()
			if jsonOutput {
				args = append(args, "--json")
			}
			var out, stderr bytes.Buffer
			if err := Execute(args, nil, &out, &stderr, "test"); err != nil || stderr.Len() != 0 {
				t.Fatalf("%v: %v %s", args, err, &stderr)
			}
			if jsonOutput && !json.Valid(out.Bytes()) {
				t.Fatalf("invalid JSON: %s", &out)
			}
			return out.String()
		}
		humans := run("help", "humans")
		if run("help") != humans || run("--help") != humans {
			t.Fatal("default help differs from human help")
		}
		if strings.Contains(humans, "Machine configuration") || !strings.Contains(humans, "means help agents") {
			t.Fatal("human help has the wrong audience")
		}
		agents := run("help", "agents")
		if run("agents") != agents || !strings.Contains(agents, "Machine configuration") {
			t.Fatal("agent guide or compatibility alias is incomplete")
		}
		banner := run()
		for _, topic := range []string{"means help", "means help agents"} {
			if !strings.Contains(banner, topic) {
				t.Fatalf("banner omits %q: %s", topic, banner)
			}
		}
		if strings.Contains(banner, "--path") || strings.Contains(banner, "Get started") || !strings.Contains(banner, "init") {
			t.Fatalf("unexpected banner: %s", banner)
		}
		if run("help", "schema", "set") != run("schema", "set", "--help") {
			t.Fatal("nested command help differs")
		}
	}
}

func TestAgentGuideDiscovery(t *testing.T) {
	for _, command := range [][]string{{"--help"}, {"init", "--help"}, {"agents", "--help"}} {
		var out, stderr bytes.Buffer
		if err := Execute(append(command, "--json"), nil, &out, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		if !json.Valid(out.Bytes()) || !strings.Contains(out.String(), "means help agents") {
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
			if err := json.Unmarshal(out.Bytes(), &result); err != nil || len(result.NextSteps) != 1 || result.NextSteps[0] != "means help agents" {
				t.Fatalf("next steps: %s %v", &out, err)
			}
		} else if !strings.Contains(out.String(), "means help agents") {
			t.Fatal("missing next step")
		}
	}
}
