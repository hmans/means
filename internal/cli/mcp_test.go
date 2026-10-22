package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hmans/means/internal/collection"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestMCPToolsOverStdio(t *testing.T) {
	for _, protocol := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(protocol, func(t *testing.T) {
			path := t.TempDir()
			c, err := collection.Init(path, []byte(`{"type":"object","properties":{"title":{"type":"string"},"count":{"type":"integer"},"ratio":{"type":"number"}},"additionalProperties":false}`))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			session := connectMCP(t, ctx, path, protocol)
			tools, err := session.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(tools.Tools) != 8 {
				t.Fatalf("tools = %v", tools.Tools)
			}
			foundSearch := false
			for _, tool := range tools.Tools {
				if tool.Name == "search" {
					foundSearch = true
				}
				if tool.Description == "" || tool.InputSchema == nil || tool.Annotations == nil {
					t.Fatalf("missing tool metadata: %+v", tool)
				}
				if tool.Annotations.ReadOnlyHint != (tool.Name != "create" && tool.Name != "update" && tool.Name != "delete" && tool.Name != "schema_set") {
					t.Fatalf("incorrect read-only annotation: %+v", tool)
				}
				if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint != (tool.Name == "update" || tool.Name == "delete" || tool.Name == "schema_set") {
					t.Fatalf("incorrect destructive annotation: %+v", tool)
				}
			}
			if !foundSearch {
				t.Fatal("search tool missing")
			}
			call := func(name, args string, wantError bool) map[string]any {
				t.Helper()
				result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: json.RawMessage(args)})
				if err != nil {
					t.Fatalf("%s protocol error: %v", name, err)
				}
				if result.IsError != wantError {
					t.Fatalf("%s(%s) = %+v, want isError %v", name, args, result, wantError)
				}
				if wantError {
					if len(result.Content) == 0 {
						t.Fatal("tool error has no explanation")
					}
					return nil
				}
				if result.StructuredContent == nil || len(result.Content) != 1 {
					t.Fatalf("missing structured/text output: %+v", result)
				}
				text, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatalf("unexpected content: %T", result.Content[0])
				}
				var compact bytes.Buffer
				if err := json.Compact(&compact, []byte(text.Text)); err != nil || compact.String() != text.Text {
					t.Fatalf("text is not compact JSON: %+v", result)
				}
				value, err := jsonschema.UnmarshalJSON(strings.NewReader(text.Text))
				if err != nil {
					t.Fatal(err)
				}
				return value.(map[string]any)
			}
			info := call("info", `{}`, false)
			if info["path"] != path || info["schema"] == nil || info["instructions"] == "" {
				t.Fatalf("collection info = %v", info)
			}
			for _, args := range []string{
				`{}`, `{"frontmatter":null}`, `{"frontmatter":[]}`,
				`{"frontmatter":{"title":42}}`, `{"frontmatter":{"title":"x"},"body":null}`,
				`{"frontmatter":{"title":"x"},"path":"/tmp"}`,
				`{"frontmatter":{"id":"supplied","title":"x"}}`,
				`{"frontmatter":{"title":"x","data":null}}`,
				`{"frontmatter":{"title":"x","data":{"count":"wrong"}}}`,
				`{"frontmatter":{"title":"x","count":1}}`,
			} {
				call("create", args, true)
			}
			files, err := filepath.Glob(filepath.Join(path, "*.md"))
			if err != nil || len(files) != 0 {
				t.Fatalf("invalid calls created files: %v, %v", files, err)
			}
			created := call("create", `{"frontmatter":{"title":"Precise numbers","data":{"count":9007199254740993,"ratio":0.1234567890123456789}},"body":"## Notes\n\nExact body.\n"}`, false)
			id := created["id"].(string)
			if created["frontmatter"].(map[string]any)["id"] != id {
				t.Fatal("missing stored ID")
			}
			// MCP resolves stored IDs after moving the actual file, too.
			if err := os.Mkdir(filepath.Join(path, "nested"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(filepath.Join(path, id+"-precise-numbers.md"), filepath.Join(path, "nested", "renamed.md")); err != nil {
				t.Fatal(err)
			}
			read := call("read", `{"id":"`+id+`"}`, false)
			metadata := read["frontmatter"].(map[string]any)["data"].(map[string]any)
			if metadata["count"] != json.Number("9007199254740993") || metadata["ratio"] != json.Number("0.1234567890123456789") || read["body"] != "## Notes\n\nExact body.\n" {
				t.Fatalf("round-trip lost content: %v", read)
			}
			hash := read["hash"].(string)
			if hash != created["hash"] || len(hash) != 64 {
				t.Fatal("missing or inconsistent hash")
			}
			call("update", `{"id":"`+id+`","frontmatter":{"title":"Changed"}}`, true)
			call("update", `{"id":"`+id+`","hash":"`+hash+`"}`, true)
			call("update", `{"id":"`+id+`","hash":"`+hash+`","body":null}`, true)
			call("update", `{"id":"`+id+`","hash":"`+hash+`","frontmatter":{"title":null}}`, true)
			for _, fm := range []string{`{"id":"` + id + `"}`, `{"id":null}`, `{"data":null}`, `{"data":{"ratio":"wrong"}}`, `{"status":"done"}`} {
				call("update", `{"id":"`+id+`","hash":"`+hash+`","frontmatter":`+fm+`}`, true)
			}
			updated := call("update", `{"id":"`+id+`","hash":"`+hash+`","frontmatter":{"title":"Changed","data":{"count":null}}}`, false)
			if updated["hash"] == hash || updated["body"] != read["body"] || updated["frontmatter"].(map[string]any)["data"].(map[string]any)["ratio"] != metadata["ratio"] {
				t.Fatalf("unexpected patch: %v", updated)
			}
			if _, exists := updated["frontmatter"].(map[string]any)["data"].(map[string]any)["count"]; exists {
				t.Fatal("null did not remove count")
			}
			call("update", `{"id":"`+id+`","hash":"`+hash+`","body":"stale"}`, true)
			updated = call("update", `{"id":"`+id+`","hash":"`+updated["hash"].(string)+`","body":""}`, false)
			if updated["body"] != "" {
				t.Fatal("body not cleared")
			}
			for range 20 {
				if _, err := c.Create(map[string]any{"title": "Another document"}, ""); err != nil {
					t.Fatal(err)
				}
			}
			listed := call("list", `{}`, false)
			if len(listed["documents"].([]any)) != 20 || listed["has_more"] != true {
				t.Fatalf("default limit: %v", listed)
			}
			if err := os.WriteFile(filepath.Join(path, "broken.md"), []byte("no frontmatter"), 0o644); err != nil {
				t.Fatal(err)
			}
			listed = call("list", `{"limit":0}`, false)
			if len(listed["documents"].([]any)) != 22 || listed["has_more"] != false {
				t.Fatalf("unlimited list: %v", listed)
			}
			foundBroken := false
			for _, entry := range listed["documents"].([]any) {
				doc := entry.(map[string]any)
				if _, exists := doc["body"]; exists {
					t.Fatal("list includes body")
				}
				if doc["path"] == "broken.md" {
					foundBroken = doc["error"] != nil && doc["frontmatter"] == nil
				}
			}
			if !foundBroken {
				t.Fatal("missing malformed-document error")
			}
			call("read", `{"id":"../outside"}`, true)
			call("read", `{"id":"missing"}`, true)
			call("list", `{"limit":-1}`, true)
			call("list", `{"limit":0.5}`, true)
			call("list", `{"limit":1e100}`, true)
			call("list", `{"limit":1e0}`, false)
			call("info", `{"path":"elsewhere"}`, true)
			call("info", `null`, true)
			// Schema changes affect the next call; reads still allow old metadata.
			if err := os.WriteFile(filepath.Join(path, "means.schema.json"), []byte(`{"type":"object","required":["category"]}`), 0o644); err != nil {
				t.Fatal(err)
			}
			info = call("info", `{}`, false)
			if info["schema"].(map[string]any)["required"].([]any)[0] != "category" {
				t.Fatal("stale schema")
			}
			call("create", `{"frontmatter":{"title":"Old schema"}}`, true)
			created = call("create", `{"frontmatter":{"title":"New schema","data":{"category":"new"}}}`, false)
			if created["body"] != "" {
				t.Fatal("body did not default to empty")
			}
			call("read", `{"id":"`+id+`"}`, false)
			// Delete validates its inputs and hash, but not existing application data.
			for _, args := range []string{
				`{}`, `{"id":"` + id + `"}`, `{"id":"` + id + `","hash":null}`,
				`{"id":"` + id + `","hash":"bad"}`,
				`{"id":"` + id + `","hash":"` + hash + `"}`,
				`{"id":"` + id + `","hash":"` + updated["hash"].(string) + `","force":true}`,
			} {
				call("delete", args, true)
			}
			deleteArgs := `{"id":"` + id + `","hash":"` + updated["hash"].(string) + `"}`
			deleted := call("delete", deleteArgs, false)
			if len(deleted) != 1 || deleted["id"] != id {
				t.Fatalf("delete output: %v", deleted)
			}
			call("delete", deleteArgs, true)
			call("read", `{"id":"`+id+`"}`, true)
			call("read", `{"id":"`+created["id"].(string)+`"}`, false)
			if err := os.WriteFile(filepath.Join(path, "means.schema.json"), []byte(`{"type":42}`), 0o644); err != nil {
				t.Fatal(err)
			}
			call("info", `{}`, true)
			if err := os.WriteFile(filepath.Join(path, "means.schema.json"), []byte(`true`), 0o644); err != nil {
				t.Fatal(err)
			}
			if call("info", `{}`, false)["schema"] != true {
				t.Fatal("boolean schema not preserved")
			}
			fixture, err := c.Create(map[string]any{"title": "Search fixture"}, "Timeout during deployment")
			if err != nil {
				t.Fatal(err)
			}
			full := call("search", `{"query":"deployment timeout"}`, false)
			if len(full["documents"].([]any)) != 1 {
				t.Fatalf("full text: %v", full)
			}
			literal := call("search", `{"query":"deployment timeout","literal":true}`, false)
			if len(literal["documents"].([]any)) != 0 {
				t.Fatalf("literal: %v", literal)
			}
			call("search", `{"query":"timeout","literal":"yes"}`, true)
			// External edits are observed by the next call on the same connection.
			changed := strings.Replace(fixture.Markdown, "Timeout", "Latency", 1)
			if err := os.WriteFile(filepath.Join(path, fixture.ID+"-search-fixture.md"), []byte(changed), 0o644); err != nil {
				t.Fatal(err)
			}
			if len(call("search", `{"query":"deployment timeout"}`, false)["documents"].([]any)) != 0 {
				t.Fatal("stale MCP result")
			}
			if len(call("search", `{"query":"deployment latency"}`, false)["documents"].([]any)) != 1 {
				t.Fatal("external edit not indexed")
			}
		})
	}
}

// Exercise Cobra and actual newline-delimited IO, including --json and EOF.
func connectMCP(t *testing.T, ctx context.Context, path, protocol string, extraFlags ...string) *mcp.ClientSession {
	t.Helper()
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	var wire, stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		args := append([]string{"mcp", "--path", path}, extraFlags...)
		done <- Execute(args, serverIn,
			io.MultiWriter(serverOut, &wire), &stderr, "test-mcp")
	}()
	t.Cleanup(func() {
		clientOut.Close()
		defer clientIn.Close()
		defer serverIn.Close()
		defer serverOut.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("MCP exit: %v, stderr: %s", err, stderr.String())
			}
		case <-time.After(3 * time.Second):
			t.Fatal("MCP did not exit after disconnect")
		}
		scanner := bufio.NewScanner(&wire)
		preciseStructuredContent := false
		for scanner.Scan() {
			if !json.Valid(scanner.Bytes()) {
				t.Errorf("stdout has a non-MCP line: %q", scanner.Text())
			}
			// Check the wire, since client libraries may themselves decode into
			// float64. The server must preserve numbers in structuredContent too.
			var response struct {
				Result struct {
					StructuredContent json.RawMessage `json:"structuredContent"`
				} `json:"result"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &response); err == nil &&
				bytes.Contains(response.Result.StructuredContent, []byte(`9007199254740993`)) &&
				bytes.Contains(response.Result.StructuredContent, []byte(`0.1234567890123456789`)) {
				preciseStructuredContent = true
			}
		}
		if err := scanner.Err(); err != nil {
			t.Error(err)
		}
		if !preciseStructuredContent {
			t.Error("precise numbers missing from structured wire output")
		}
	})
	client := mcp.NewClient(&mcp.Implementation{Name: "means-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.IOTransport{Reader: clientIn, Writer: clientOut},
		&mcp.ClientSessionOptions{ProtocolVersion: protocol})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestMCPJSONStructuredOnly(t *testing.T) {
	for _, protocol := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(protocol, func(t *testing.T) {
			path := t.TempDir()
			if _, err := collection.Init(path, []byte(`{"type":"object","properties":{"count":{"type":"integer"},"ratio":{"type":"number"}}}`)); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			session := connectMCP(t, ctx, path, protocol, "--json")
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "create", Arguments: json.RawMessage(`{"frontmatter":{"title":"Precise numbers","data":{"count":9007199254740993,"ratio":0.1234567890123456789}}}`)})
			if err != nil || result.IsError || result.StructuredContent == nil || len(result.Content) != 0 {
				t.Fatalf("structured-only result = %+v, %v", result, err)
			}
			failure, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "read", Arguments: json.RawMessage(`{"id":"missing"}`)})
			if err != nil || !failure.IsError || len(failure.Content) == 0 {
				t.Fatalf("error lost text = %+v, %v", failure, err)
			}
		})
	}
}

func TestMCPStartupAndHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"mcp", "--path", t.TempDir(), "--json"}, strings.NewReader(""), &stdout, &stderr, "test"); err == nil {
		t.Fatal("missing schema did not fail startup")
	}
	if stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"mcp", "--help", "--json"}, nil, &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	var help map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &help); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(help["description"], "info") || !strings.Contains(help["description"], "--json omits") ||
		!strings.Contains(help["description"], "framing") || strings.Contains(help["usage"], "--structured-only") {
		t.Fatalf("incomplete MCP help: %v", help)
	}
	path := t.TempDir()
	if _, err := collection.Init(path, nil); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"mcp", "--path", path}, strings.NewReader(""), &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("EOF produced output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
