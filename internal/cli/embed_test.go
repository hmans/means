package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hmans/means/internal/collection"
)

func TestEmbeddingCLI(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0]}]}`))
	}))
	defer server.Close()
	t.Setenv("MEANS_EMBEDDING_BASE_URL", server.URL)
	t.Setenv("MEANS_EMBEDDING_API_KEY", "test")
	c, err := embeddingTestCollection(t, t.TempDir(), "{provider: openai, dimensions: 2}")
	if err != nil {
		t.Fatal(err)
	}
	var progress string
	run := func(input string, args ...string) []byte {
		t.Helper()
		var out, stderr bytes.Buffer
		args = append(args, "--path", c.Path, "--json")
		if err := Execute(args, strings.NewReader(input), &out, &stderr, "test"); err != nil {
			t.Fatalf("%v: %v %s", args, err, stderr.String())
		}
		progress = stderr.String()
		if !json.Valid(out.Bytes()) {
			t.Fatalf("progress contaminated JSON: %s", out.String())
		}
		if progress != "" {
			t.Fatalf("JSON mode emitted progress: %s", progress)
		}
		return out.Bytes()
	}
	var doc collection.Document
	if err := json.Unmarshal(run(`{"frontmatter":{"title":"original"},"body":"body"}`, "create", "--input", "-"), &doc); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("CLI create did not embed")
	}
	var result collection.SearchResult
	if err := json.Unmarshal(run("", "search", "meaning", "--semantic", "--with-body"), &result); err != nil || len(result.Documents) != 1 || result.Documents[0].Score == nil || result.Documents[0].Body == nil {
		t.Fatalf("semantic CLI result: %+v %v", result, err)
	}
	refreshed := run("", "embed")
	var textOut, textErr bytes.Buffer
	if err := Execute([]string{"embed", "--path", c.Path}, nil, &textOut, &textErr, "test"); err != nil || !strings.Contains(textErr.String(), "0 generated, 1 reused, 0 removed") {
		t.Fatalf("text refresh progress: %s %v", textErr.String(), err)
	}
	if bytes.Contains(refreshed, []byte(`"body"`)) || bytes.Contains(refreshed, []byte(`"vector"`)) || calls.Load() != 2 {
		t.Fatalf("embed output/calls: %s / %d", refreshed, calls.Load())
	}
	run("", "embed", "--force")
	if calls.Load() != 3 {
		t.Fatalf("force did not regenerate: %d %s", calls.Load(), progress)
	}
	patch, _ := json.Marshal(map[string]any{"hash": doc.Hash, "frontmatter": map[string]any{"title": "changed"}})
	if err := json.Unmarshal(run(string(patch), "update", doc.ID, "--input", "-"), &doc); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 4 {
		t.Fatal("CLI update did not embed")
	}
	run(`{"hash":"`+doc.Hash+`"}`, "delete", doc.ID, "--input", "-")
	if calls.Load() != 4 {
		t.Fatal("CLI delete called API")
	}
	cache, err := os.ReadFile(filepath.Join(c.Path, ".means", "embeddings.jsonl"))
	if err != nil || bytes.Contains(cache, []byte(doc.ID)) {
		t.Fatalf("delete retained embedding: %s %v", cache, err)
	}
	for _, args := range [][]string{{"search", "--semantic"}, {"search", "text", "--semantic", "--literal"}} {
		var out, stderr bytes.Buffer
		if err := Execute(append(args, "--path", c.Path, "--json"), nil, &out, &stderr, "test"); err == nil || out.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Fatalf("invalid args %v: %v %s", args, err, stderr.String())
		}
	}
}
