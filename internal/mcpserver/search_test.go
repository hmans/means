package mcpserver

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/hmans/means/internal/collection"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSearchTool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	path := t.TempDir()
	if _, err := collection.Init(path, []byte(`{"type":"object"}`)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(path, "a.md"), "---\nid: alpha\ntitle: Schema work\ndata:\n  type: feature\n  priority: normal\n  count: 9007199254740993\n---\nbody")
	writeFile(t, filepath.Join(path, "b.md"), "---\nid: beta\ntitle: Other\ndata:\n  type: task\n  priority: high\n---\nbody")
	writeFile(t, filepath.Join(path, "bad.md"), "broken")
	server, err := New(path, "test")
	if err != nil {
		t.Fatal(err)
	}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	result := callTool(t, ctx, session, "search", `{"query":"SCHEMA","filter":{"type":"feature","priority":"normal","count":9007199254740993},"limit":0}`)
	docs := result["documents"].([]any)
	if len(docs) != 1 || docs[0].(map[string]any)["id"] != "alpha" || result["has_more"] != false || len(result["diagnostics"].([]any)) != 1 {
		t.Fatalf("search result: %#v", result)
	}
	if docs[0].(map[string]any)["match"].(map[string]any)["field"] != "title" {
		t.Fatalf("missing match context: %#v", docs[0])
	}
	result = callTool(t, ctx, session, "search", `{"filter":{"type":"missing"}}`)
	if len(result["documents"].([]any)) != 0 || len(result["diagnostics"].([]any)) != 1 {
		t.Fatalf("empty result lost diagnostics: %#v", result)
	}
	bad, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search", Arguments: json.RawMessage(`{"limit":-1}`)})
	if err != nil || !bad.IsError {
		t.Fatalf("invalid limit accepted: %+v %v", bad, err)
	}
}
