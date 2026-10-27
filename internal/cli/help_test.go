package cli

import (
	"bytes"
	"encoding/json"
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
		if flag != topic || flag != legacy {
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
	for _, alias := range [][]string{{"help", "humans"}, {"help", "agents"}, {"agents"}} {
		if run(alias...) != run("help") {
			t.Fatalf("alias differs: %v", alias)
		}
	}
	banner := run()
	if !strings.Contains(banner, "means help mcp") || strings.Contains(banner, "means help agents") || strings.Contains(banner, "--path") {
		t.Fatal(banner)
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
