package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSharedCommandHelp(t *testing.T) {
	run := func(args ...string) string {
		t.Helper()
		var out, stderr bytes.Buffer
		if err := Execute(args, nil, &out, &stderr, "test"); err != nil || stderr.Len() != 0 {
			t.Fatalf("%v: %v %s", args, err, &stderr)
		}
		return out.String()
	}
	root, _ := newRootCommand("test")
	var check func(*cobra.Command)
	check = func(cmd *cobra.Command) {
		path := strings.Fields(strings.TrimPrefix(cmd.CommandPath(), "means "))
		if cmd == root {
			path = nil
		}
		flag := run(append(append([]string{}, path...), "--help")...)
		topic := run(append([]string{"help"}, path...)...)
		legacy := run(append([]string{"help", "agents"}, path...)...)
		if flag != topic || (cmd != root && flag != legacy) {
			t.Fatalf("help routes differ for %v", path)
		}
		var result struct{ Description, Usage string }
		if err := json.Unmarshal([]byte(run(append(append([]string{}, path...), "--help", "--json")...)), &result); err != nil {
			t.Fatal(err)
		}
		if flag != result.Description+"\n\n"+result.Usage {
			t.Fatalf("JSON has different help for %v", path)
		}
		for _, child := range cmd.Commands() {
			if child.Name() != "agents" && child.Name() != "help" {
				check(child)
			}
		}
	}
	check(root)
	for _, alias := range [][]string{{"help", "humans"}} {
		if run(alias...) != run("help") {
			t.Fatalf("alias differs: %v", alias)
		}
	}
	banner := run()
	if !strings.Contains(banner, "means help mcp") || !strings.Contains(banner, "means help agents") || strings.Contains(banner, "--path") {
		t.Fatal(banner)
	}
}

func TestAgentsGuide(t *testing.T) {
	workspace, config := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	run := func(args ...string) string {
		t.Helper()
		var out, stderr bytes.Buffer
		args = append(args, "--path", workspace)
		if err := Execute(args, nil, &out, &stderr, "test"); err != nil || stderr.Len() != 0 {
			t.Fatalf("%v: %v %s", args, err, &stderr)
		}
		return out.String()
	}
	text := run("help", "agents")
	if text != run("help", "agents", "--help") {
		t.Fatal("guide help flag differs")
	}
	for _, args := range [][]string{{"help", "agents", "--json"}, {"agents", "--json"}} {
		var result struct{ Command, Description, Usage string }
		if err := json.Unmarshal([]byte(run(args...)), &result); err != nil {
			t.Fatal(err)
		}
		if result.Description != strings.TrimSpace(agentsGuide) {
			t.Fatal("guide content differs")
		}
		if args[0] == "help" && text != result.Description+"\n\n"+result.Usage {
			t.Fatal("text and JSON differ")
		}
	}
	for _, dir := range []string{workspace, config} {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("help changed %s: %v %v", dir, entries, err)
		}
	}

	// Validate the actual embedded examples together, so format changes cannot
	// silently leave the installed guide with an invalid collection or document.
	blocks := strings.Split(agentsGuide, "```yaml\n")
	if len(blocks) != 3 {
		t.Fatal("expected configuration and document examples")
	}
	for i, name := range []string{"MEANS.md", "a1b2c-use-a-single-binary.md"} {
		source := strings.SplitN(blocks[i+1], "```", 2)[0]
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var report struct {
		Healthy bool `json:"healthy"`
		Checked int  `json:"checked"`
	}
	if err := json.Unmarshal([]byte(run("doctor", "--json")), &report); err != nil || !report.Healthy || report.Checked != 1 {
		t.Fatalf("invalid guide examples: %+v %v", report, err)
	}
}

func TestInitHelpDiscovery(t *testing.T) {
	var out, stderr bytes.Buffer
	if err := Execute([]string{"init", "--path", filepath.Join(t.TempDir(), "memory"), "--json"}, nil, &out, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	var result struct {
		NextSteps []string `json:"next_steps"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || len(result.NextSteps) != 1 || result.NextSteps[0] != "means help create" {
		t.Fatalf("%s %v", &out, err)
	}
}
