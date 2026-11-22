package mcpserver

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hmans/means/collection"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSearchTool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	path := t.TempDir()
	if _, err := collection.Init(path, []byte(`{"type":"object"}`)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(path, "a.md"), "---\nid: alpha\ntitle: Schema work\ndata:\n  type: feature\n  priority: normal\n  count: 9007199254740993\n  at: 2026-10-26T10:00:00+01:00\n---\nbody")
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
	for _, tool := range []string{"list", "search"} {
		sorted := callTool(t, ctx, session, tool, `{"sort":["-data.priority"],"limit":1}`)
		if docs := sorted["documents"].([]any); len(docs) != 1 || docs[0].(map[string]any)["id"] != "alpha" || sorted["has_more"] != true {
			t.Fatalf("%s sort: %#v", tool, sorted)
		}
	}
	if result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list", Arguments: map[string]any{"sort": []any{"body"}}}); err != nil || !result.IsError {
		t.Fatalf("invalid sort key: %+v %v", result, err)
	}
	result = callTool(t, ctx, session, "search", `{"filter":{"type":"missing"}}`)
	if len(result["documents"].([]any)) != 0 || len(result["diagnostics"].([]any)) != 1 {
		t.Fatalf("empty result lost diagnostics: %#v", result)
	}
	result = callTool(t, ctx, session, "search", `{"datetime":[{"field":"at","op":"on_or_after","value":"2026-10-26T09:00:00Z"},{"field":"at","op":"before","value":"2026-10-27T00:00:00Z"}],"filter":{"type":"feature"},"limit":1}`)
	if docs := result["documents"].([]any); len(docs) != 1 || docs[0].(map[string]any)["id"] != "alpha" || result["has_more"] != false {
		t.Fatalf("datetime: %#v", result)
	}
	for _, input := range []string{
		`{"datetime":[{"field":"at","op":"before","value":"invalid"}]}`,
		`{"datetime":[{"field":"at","op":"unknown","value":"2026-10-26T09:00:00Z"}]}`,
		`{"datetime":[{"field":"at","op":"before"}]}`,
		`{"datetime":[{"field":"at","op":"before","value":42}]}`,
	} {
		bad, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search", Arguments: json.RawMessage(input)})
		if err != nil || !bad.IsError {
			t.Fatalf("accepted %s: %+v %v", input, bad, err)
		}
	}
	bad, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search", Arguments: json.RawMessage(`{"limit":-1}`)})
	if err != nil || !bad.IsError {
		t.Fatalf("invalid limit accepted: %+v %v", bad, err)
	}
}

func TestViewArgumentAndInfo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	path := t.TempDir()
	if _, err := collection.Init(path, []byte(`{"type":"object"}`)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(path, collection.ConfigFilename), "---\nschema: {type: object}\napp:\n  views:\n    - title: Open\n      filter: data.status == \"todo\"\n      columns: [title]\n---\n")
	writeFile(t, filepath.Join(path, "a.md"), "---\nid: alpha\ntitle: Schema work\ndata: {status: todo, priority: high}\n---\nbody")
	writeFile(t, filepath.Join(path, "b.md"), "---\nid: beta\ntitle: Schema other\ndata: {status: done, priority: high}\n---\nbody")
	writeFile(t, filepath.Join(path, "c.md"), "---\nid: gamma\ntitle: Schema low\ndata: {status: todo, priority: low}\n---\nbody")
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
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	info := callTool(t, ctx, session, "info", `{}`)
	views := info["views"].([]any)
	if len(views) != 1 || views[0].(map[string]any)["title"] != "Open" || views[0].(map[string]any)["filter"] != `data.status == "todo"` {
		t.Fatalf("info views: %#v", info["views"])
	}
	for _, args := range []string{
		`{"view":"Open","where":"data.priority == \"high\""}`,
		`{"view":"Open","filter":{"priority":"high"},"query":"schema"}`,
	} {
		tool := "list"
		if strings.Contains(args, "query") {
			tool = "search"
		}
		result := callTool(t, ctx, session, tool, args)
		if docs := result["documents"].([]any); len(docs) != 1 || docs[0].(map[string]any)["id"] != "alpha" {
			t.Fatalf("%s %s: %#v", tool, args, result)
		}
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search", Arguments: map[string]any{"view": "Missing"}})
	if err != nil || !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, `available views: "Open"`) {
		t.Fatalf("unknown view: %+v %v", result, err)
	}
}
