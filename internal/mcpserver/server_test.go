package mcpserver

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

func TestMultipleCollectionDescriptions(t *testing.T) {
	for _, protocol := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(protocol, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			t.Cleanup(cancel)
			sessions := make(map[string]*mcp.ClientSession)
			paths := make(map[string]string)
			for _, title := range []string{"Issues", "Decisions"} {
				path := t.TempDir()
				schema, _ := json.Marshal(map[string]any{"title": title, "description": "Stores " + title, "type": "object"})
				if _, err := collection.Init(path, schema); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(path, collection.InstructionsFilename), "Workflow for "+title)
				server, err := New(path, "test")
				if err != nil {
					t.Fatal(err)
				}
				clientTransport, serverTransport := mcp.NewInMemoryTransports()
				ss, err := server.Connect(ctx, serverTransport, nil)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { ss.Close() })
				client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
				session, err := client.Connect(ctx, clientTransport, &mcp.ClientSessionOptions{ProtocolVersion: protocol})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { session.Close() })
				sessions[title], paths[title] = session, path
				initialized := session.InitializeResult()
				if initialized.ServerInfo.Name != "means" || initialized.ServerInfo.Title != title || initialized.ServerInfo.Description != "Stores "+title ||
					!strings.Contains(initialized.Instructions, "Stores "+title) || !strings.Contains(initialized.Instructions, "Workflow for "+title) {
					t.Fatalf("missing collection identity/workflow at discovery: %+v", initialized)
				}
				tools, err := session.ListTools(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				for _, tool := range tools.Tools {
					if !strings.HasPrefix(tool.Description, "Collection: "+title+".") || !strings.Contains(tool.Description, "Stores "+title) {
						t.Fatalf("tool is missing collection purpose: %+v", tool)
					}
				}
			}
			// The same client can call identical tool names on distinct sessions.
			issue := callTool(t, ctx, sessions["Issues"], "create_document", `{"frontmatter":{"title":"A bug"}}`)
			id := issue["id"].(string)
			result, err := sessions["Decisions"].CallTool(ctx, &mcp.CallToolParams{Name: "read_document", Arguments: map[string]any{"id": id}})
			if err != nil || !result.IsError {
				t.Fatalf("read crossed collection boundary: %+v %v", result, err)
			}
			for title, session := range sessions {
				info := callTool(t, ctx, session, "collection_info", `{}`)
				if info["title"] != title || info["description"] != "Stores "+title || !strings.Contains(info["instructions"].(string), "Workflow for "+title) {
					t.Fatalf("info = %v", info)
				}
				list := callTool(t, ctx, session, "list_documents", `{"limit":0}`)
				want := 0
				if title == "Issues" {
					want = 1
				}
				if len(list["documents"].([]any)) != want {
					t.Fatalf("collection %s leaked documents or instructions: %v", title, list)
				}
			}
			// Discovery is a startup snapshot; collection_info always reads current guidance.
			writeFile(t, filepath.Join(paths["Issues"], collection.SchemaFilename), `{"title":"Backlog","description":"Updated purpose","type":"object"}`)
			writeFile(t, filepath.Join(paths["Issues"], collection.InstructionsFilename), "Updated workflow")
			info := callTool(t, ctx, sessions["Issues"], "collection_info", `{}`)
			if info["title"] != "Backlog" || info["description"] != "Updated purpose" || !strings.Contains(info["instructions"].(string), "Updated workflow") {
				t.Fatalf("stale info: %v", info)
			}
			tools, err := sessions["Issues"].ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(tools.Tools[0].Description, "Collection: Issues.") {
				t.Fatal("discovery unexpectedly changed")
			}
			if err := os.Remove(filepath.Join(paths["Issues"], collection.InstructionsFilename)); err != nil {
				t.Fatal(err)
			}
			info = callTool(t, ctx, sessions["Issues"], "collection_info", `{}`)
			if strings.Contains(info["instructions"].(string), "Updated workflow") {
				t.Fatal("removed workflow retained")
			}
		})
	}
}

func callTool(t *testing.T, ctx context.Context, session *mcp.ClientSession, name, args string) map[string]any {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: json.RawMessage(args)})
	if err != nil || result.IsError {
		t.Fatalf("%s: %+v %v", name, result, err)
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
