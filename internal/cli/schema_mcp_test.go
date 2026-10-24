package cli

import (
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

func TestMCPSchemaSet(t *testing.T) {
	for _, protocol := range []string{"2025-11-25", "2026-07-28"} {
		for _, jsonFlag := range []bool{false, true} {
			t.Run(protocol+map[bool]string{true: "/json-flag", false: "/default"}[jsonFlag], func(t *testing.T) {
				c, err := collection.Init(t.TempDir(), nil)
				if err != nil {
					t.Fatal(err)
				}
				original, err := os.ReadFile(filepath.Join(c.Path, collection.ConfigFilename))
				if err != nil {
					t.Fatal(err)
				}
				doc, err := c.Create(map[string]any{"title": "Keep"}, "Body")
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
				call := func(args string, wantError bool) *mcp.CallToolResult {
					t.Helper()
					r, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "schema_set", Arguments: json.RawMessage(args)})
					if err != nil || r.IsError != wantError {
						t.Fatalf("schema_set %s: %+v %v", args, r, err)
					}
					if wantError && len(r.Content) == 0 {
						t.Fatal("error missing text")
					}
					if !wantError && len(r.Content) != 0 {
						t.Fatalf("wrong content mode: %+v", r)
					}
					return r
				}
				decode := func(r *mcp.CallToolResult) map[string]any {
					t.Helper()
					b, err := json.Marshal(r.StructuredContent)
					if err != nil {
						t.Fatal(err)
					}
					var result map[string]any
					if err := json.Unmarshal(b, &result); err != nil || result == nil {
						t.Fatalf("missing structured result: %s %v", b, err)
					}
					return result
				}
				for _, args := range []string{`{}`, `{"schema":null}`, `{"schema":"true"}`, `{"schema":[],"force":true}`, `{"schema":true,"force":"yes"}`, `{"schema":true,"path":"elsewhere"}`, `{"schema":{"type":42},"force":true}`, `{"schema":{"$ref":"other.json"},"force":true}`} {
					call(args, true)
				}
				rejected := decode(call(`{"schema":false}`, true))
				if rejected["installed"] != false || rejected["error"] == nil || len(rejected["diagnostics"].([]any)) != 1 {
					t.Fatalf("rejection: %v", rejected)
				}
				diagnostic := rejected["diagnostics"].([]any)[0].(map[string]any)
				if diagnostic["id"] != doc.ID || diagnostic["path"] == "" {
					t.Fatalf("diagnostic: %v", diagnostic)
				}
				filename := filepath.Join(c.Path, collection.ConfigFilename)
				stored, err := os.ReadFile(filename)
				if err != nil || string(stored) != string(original) {
					t.Fatal("rejection changed schema", err)
				}
				forced := decode(call(`{"schema":false,"force":true}`, false))
				if forced["installed"] != true || forced["healthy"] != false || len(forced["diagnostics"].([]any)) != 1 {
					t.Fatalf("force: %v", forced)
				}
				current, err := c.Read(doc.ID)
				if err != nil || current.Hash != doc.Hash {
					t.Fatal("document changed", err)
				}
				// A running server can repair an external invalid-schema edit.
				if err := os.WriteFile(filename, []byte("---\ntitle: Updated\nschema: {type: invalid}\n---\nUpdated workflow"), 0o644); err != nil {
					t.Fatal(err)
				}
				precise := `{"schema":{"title":"Schema annotation only","properties":{"n":{"const":9007199254740993},"ratio":{"const":0.1234567890123456789}}},"force":false}`
				installed := decode(call(precise, false))
				if installed["installed"] != true || installed["healthy"] != true {
					t.Fatalf("install: %v", installed)
				}
				stored, err = os.ReadFile(filename)
				if err != nil || !strings.Contains(string(stored), "9007199254740993") || !strings.Contains(string(stored), "0.1234567890123456789") {
					t.Fatalf("precision lost: %s %v", stored, err)
				}
				if title, properties := strings.Index(string(stored), "  title: Schema annotation only"), strings.Index(string(stored), "  properties:"); title < 0 || properties < title {
					t.Fatalf("key order lost: %s", stored)
				}
				info, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "info", Arguments: json.RawMessage(`{}`)})
				if err != nil || info.IsError || decode(info)["title"] != "Updated" {
					t.Fatalf("stale info: %+v %v", info, err)
				}
				if session.InitializeResult().ServerInfo.Title == "Updated" {
					t.Fatal("startup metadata changed")
				}
				if err := os.WriteFile(filename, []byte("---\nschema: [\n---\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				call(`{"schema":true,"force":true}`, true)
				if err := os.WriteFile(filename, []byte("---\ntitle: Repaired\n---\nBody"), 0o644); err != nil {
					t.Fatal(err)
				}
				if decode(call(`{"schema":true}`, false))["healthy"] != true {
					t.Fatal("boolean schema failed")
				}
			})
		}
	}
}
