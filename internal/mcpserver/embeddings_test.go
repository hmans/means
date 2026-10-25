package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hmans/means/internal/collection"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestEmbeddingTools(t *testing.T) {
	var calls atomic.Int32
	var orthogonalQuery atomic.Bool
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if orthogonalQuery.Load() {
			_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[0,1]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0]}]}`))
	}))
	defer api.Close()
	t.Setenv("MEANS_EMBEDDING_BASE_URL", api.URL)
	t.Setenv("MEANS_EMBEDDING_API_KEY", "test")
	c, err := collection.InitConfig(t.TempDir(), []byte("---\nschema: true\nembeddings: {provider: openai, dimensions: 2}\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	progress := make(chan string, 30)
	server, err := NewWithEmbeddingProgress(c.Path, "test", func(message string) { progress <- message })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a, b := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, a, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	info := callTool(t, ctx, session, "info", `{}`)
	if info["embeddings"] == nil {
		t.Fatal("info omitted embedding config")
	}
	doc := callTool(t, ctx, session, "create", `{"frontmatter":{"title":"original"},"body":"body"}`)
	if calls.Load() != 1 {
		t.Fatal("MCP create did not embed")
	}
	result := callTool(t, ctx, session, "search", `{"query":"meaning","semantic":true}`)
	hits := result["documents"].([]any)
	if len(hits) != 1 || hits[0].(map[string]any)["score"] == nil || hits[0].(map[string]any)["match"] != nil {
		t.Fatalf("semantic result: %v", result)
	}
	if calls.Load() != 2 {
		t.Fatal("search should embed only the query")
	}
	patch, _ := json.Marshal(map[string]any{"id": doc["id"], "hash": doc["hash"], "body": "changed"})
	doc = callTool(t, ctx, session, "update", string(patch))
	if calls.Load() != 3 {
		t.Fatal("MCP update did not embed")
	}
	patch, _ = json.Marshal(map[string]any{"id": doc["id"], "hash": doc["hash"]})
	callTool(t, ctx, session, "delete", string(patch))
	if calls.Load() != 3 {
		t.Fatal("MCP delete called API")
	}
	var messages []string
	for len(progress) > 0 {
		messages = append(messages, <-progress)
	}
	logged := strings.Join(messages, "\n")
	if !strings.Contains(logged, "Embedding document") {
		t.Fatalf("missing progress: %s", logged)
	}
	if strings.Contains(logged, "original") || strings.Contains(logged, "meaning") {
		t.Fatalf("progress contains content: %s", logged)
	}
	bad, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search", Arguments: json.RawMessage(`{"semantic":true,"literal":true,"query":"bad"}`)})
	if err != nil || !bad.IsError {
		t.Fatalf("invalid search: %+v %v", bad, err)
	}
	callTool(t, ctx, session, "create", `{"frontmatter":{"title":"threshold test"}}`)
	orthogonalQuery.Store(true)
	for _, tt := range []struct {
		args  string
		count int
	}{
		{`{"query":"meaning","semantic":true}`, 0},
		{`{"query":"meaning","semantic":true,"min_score":0}`, 1},
		{`{"query":"meaning","semantic":true,"min_score":-1}`, 1},
	} {
		result := callTool(t, ctx, session, "search", tt.args)
		if len(result["documents"].([]any)) != tt.count {
			t.Fatalf("threshold: %v", result)
		}
	}
	for _, args := range []string{`{"query":"meaning","min_score":0}`, `{"query":"meaning","semantic":true,"min_score":2}`, `{"query":"meaning","semantic":true,"min_score":"bad"}`} {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search", Arguments: json.RawMessage(args)})
		if err != nil || !result.IsError {
			t.Fatalf("invalid threshold: %+v %v", result, err)
		}
	}
}
