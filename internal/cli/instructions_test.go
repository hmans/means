package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hmans/means/internal/collection"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestInstructionsSetCLI(t *testing.T) {
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	run := func(input string, args ...string) (string, string, error) {
		t.Helper()
		var out, stderr bytes.Buffer
		args = append(args, "--path", c.Path, "--json")
		err := Execute(args, strings.NewReader(input), &out, &stderr, "test")
		return out.String(), stderr.String(), err
	}
	info, stderr, err := run("", "info")
	if err != nil || stderr != "" {
		t.Fatal(err, stderr)
	}
	var current collection.Description
	if err := json.Unmarshal([]byte(info), &current); err != nil || current.Hash == "" {
		t.Fatal(info, err)
	}
	for _, input := range []string{"{}", "null", "[]", `{"hash":null,"instructions":""}`, `{"hash":"x","instructions":null}`, `{"hash":"x","instructions":42}`, `{"hash":"x","instructions":"","extra":true}`, `{"hash":"x","instructions":""} {}`, `{"hash":"x","instructions":""}`} {
		out, stderr, err := run(input, "instructions", "set", "--input", "-")
		if err == nil || out != "" || !json.Valid([]byte(stderr)) {
			t.Fatalf("invalid input %s: %q %q %v", input, out, stderr, err)
		}
	}
	if _, _, err := run("", "instructions", "set"); err == nil {
		t.Fatal("missing --input accepted")
	}
	for i, body := range []string{"# Updated workflow\n\nSearch first.\n", ""} {
		input, err := json.Marshal(map[string]string{"hash": current.Hash, "instructions": body})
		if err != nil {
			t.Fatal(err)
		}
		inputPath := "-"
		if i == 1 {
			inputPath = filepath.Join(t.TempDir(), "input.json")
			if err := os.WriteFile(inputPath, input, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		out, stderr, err := run(string(input), "instructions", "set", "--input", inputPath)
		if err != nil || stderr != "" {
			t.Fatal(err, stderr)
		}
		if err := json.Unmarshal([]byte(out), &current); err != nil || current.Instructions != body {
			t.Fatal(out, err)
		}
		if _, _, err := run(string(input), "instructions", "set", "--input", "-"); err == nil {
			t.Fatal("stale write accepted")
		}
		out, stderr, err = run("", "info")
		var refreshed collection.Description
		if err != nil || stderr != "" || json.Unmarshal([]byte(out), &refreshed) != nil || refreshed != current {
			t.Fatal("info did not refresh", out, stderr, err)
		}
	}
}

func TestMCPInstructionsSet(t *testing.T) {
	for _, protocol := range []string{"2025-11-25", "2026-07-28"} {
		for _, jsonFlag := range []bool{false, true} {
			t.Run(protocol+map[bool]string{false: "/default", true: "/json-flag"}[jsonFlag], func(t *testing.T) {
				prefix := "---\n# Preserve comments\ntitle: Notes\nschema: {properties: {n: {const: 9007199254740993}, ratio: {const: 0.1234567890123456789}}}\n---\n"
				c, err := collection.InitConfig(t.TempDir(), []byte(prefix+"Exact workflow\n"))
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				var flags []string
				if jsonFlag {
					flags = []string{"--json"}
				}
				session := connectMCP(t, ctx, c.Path, protocol, flags...)
				call := func(name string, args any, wantError bool) map[string]any {
					t.Helper()
					r, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
					if err != nil || r.IsError != wantError {
						t.Fatalf("%s: %+v %v", name, r, err)
					}
					if wantError {
						if len(r.Content) == 0 {
							t.Fatal("missing error explanation")
						}
						return nil
					}
					if len(r.Content) != 0 || r.StructuredContent == nil {
						t.Fatalf("expected structured-only response: %+v", r)
					}
					data, err := json.Marshal(r.StructuredContent)
					if err != nil {
						t.Fatal(err)
					}
					var result map[string]any
					if err := json.Unmarshal(data, &result); err != nil {
						t.Fatal(err)
					}
					return result
				}
				info := call("info", map[string]any{}, false)
				if info["instructions"] != "Exact workflow\n" || info["hash"] != c.Describe().Hash || info["usage"] == "" {
					t.Fatalf("info must return exact editable body: %v", info)
				}
				for _, input := range []string{`{}`, `{"hash":null,"instructions":""}`, `{"hash":"bad","instructions":""}`, `{"hash":"` + c.Describe().Hash + `","instructions":null}`, `{"hash":"` + c.Describe().Hash + `","instructions":42}`, `{"hash":"` + c.Describe().Hash + `","instructions":"","schema":true}`} {
					call("instructions_set", json.RawMessage(input), true)
				}
				for _, body := range []string{"\n# Updated workflow\n", ""} {
					args := map[string]any{"hash": info["hash"], "instructions": body}
					updated := call("instructions_set", args, false)
					call("instructions_set", args, true)
					info = call("info", map[string]any{}, false)
					if info["instructions"] != body || info["hash"] != updated["hash"] || updated["instructions"] != body {
						t.Fatal("info did not refresh", info, updated)
					}
					stored, err := os.ReadFile(filepath.Join(c.Path, collection.ConfigFilename))
					if err != nil || string(stored) != prefix+body {
						t.Fatal("frontmatter or body changed", string(stored), err)
					}
				}
				if !strings.Contains(session.InitializeResult().Instructions, "Exact workflow\n") {
					t.Fatal("startup guidance should remain a snapshot")
				}
			})
		}
	}
}
