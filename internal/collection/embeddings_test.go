package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gofrs/flock"
)

type embeddingAPI struct {
	mu            sync.Mutex
	inputs        []string
	batches       [][]string
	failOnRequest int
	status        int
	response      string
	hook          func()
}

func TestEmbeddingProviderCredentials(t *testing.T) {
	t.Setenv("MEANS_EMBEDDING_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "direct-key")
	t.Setenv("OPENROUTER_API_KEY", "router-key")
	for provider, expected := range map[string]string{
		"openai":     "direct-key",
		"openrouter": "router-key",
	} {
		key, err := embeddingAPIKey(provider)
		if err != nil || key != expected {
			t.Fatalf("wrong credential selected for %s", provider)
		}
	}
	t.Setenv("OPENROUTER_API_KEY", "")
	if _, err := embeddingAPIKey("openrouter"); err == nil || !strings.Contains(err.Error(), "OPENROUTER_API_KEY") {
		t.Fatal("OpenRouter must not fall back to an OpenAI credential")
	}
	t.Setenv("OPENROUTER_API_KEY", "router-key")
	t.Setenv("OPENAI_API_KEY", "")
	if _, err := embeddingAPIKey("openai"); err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatal("OpenAI must not fall back to an OpenRouter credential")
	}
	t.Setenv("MEANS_EMBEDDING_API_KEY", "override")
	key, err := embeddingAPIKey("openrouter")
	if err != nil || key != "override" {
		t.Fatal("explicit key override ignored")
	}
}

func TestEmbeddingProviderDefaults(t *testing.T) {
	t.Setenv("MEANS_EMBEDDING_BASE_URL", "")
	for _, tt := range []struct{ provider, model, endpoint string }{
		{"openai", "text-embedding-3-small", "https://api.openai.com/v1"},
		{"openrouter", "openai/text-embedding-3-small", "https://openrouter.ai/api/v1"},
	} {
		c, err := InitConfig(t.TempDir(), []byte("---\nschema: true\nembeddings: {provider: "+tt.provider+"}\n---\n"))
		if err != nil {
			t.Fatal(err)
		}
		cfg := c.Embeddings()
		if cfg.Provider != tt.provider || cfg.Model != tt.model || cfg.Dimensions != 512 {
			t.Fatalf("defaults: %+v", cfg)
		}
		s, err := c.openEmbeddings()
		if err != nil {
			t.Fatal(err)
		}
		s.close()
		if s.endpoint != tt.endpoint {
			t.Fatalf("endpoint: %s", s.endpoint)
		}
	}
}

func TestOpenRouterEndpointOverride(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "wrong-provider")
	t.Setenv("OPENROUTER_API_KEY", "router-key")
	t.Setenv("MEANS_EMBEDDING_API_KEY", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" || r.Header.Get("Authorization") != "Bearer router-key" {
			t.Error("custom endpoint did not retain provider credentials")
		}
		var request struct {
			Model      string
			Dimensions int
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Model != "openai/text-embedding-3-small" || request.Dimensions != 2 {
			t.Errorf("request: %+v", request)
		}
		fmt.Fprint(w, `{"data":[{"index":0,"embedding":[1,0]}]}`)
	}))
	defer server.Close()
	t.Setenv("MEANS_EMBEDDING_BASE_URL", server.URL+"/v1/")
	c, err := InitConfig(t.TempDir(), []byte("---\nschema: true\nembeddings: {provider: openrouter, dimensions: 2}\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Create(map[string]any{"title": "routed"}, "body"); err != nil {
		t.Fatal(err)
	}
}

func fakeEmbeddingAPI(t *testing.T) *embeddingAPI {
	t.Helper()
	api := &embeddingAPI{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		defer api.mu.Unlock()
		if r.Method != "POST" || r.URL.Path != "/embeddings" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("bad embedding request: %s %s", r.Method, r.URL.Path)
		}
		var request struct {
			Input      json.RawMessage
			Model      string
			Dimensions int
			Encoding   string `json:"encoding_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Model != "text-embedding-3-small" || request.Dimensions != 2 || request.Encoding != "float" {
			t.Errorf("request: %+v", request)
		}
		var inputs []string
		if err := json.Unmarshal(request.Input, &inputs); err != nil {
			var input string
			if err := json.Unmarshal(request.Input, &input); err != nil {
				t.Error(err)
			}
			inputs = []string{input}
		}
		api.inputs = append(api.inputs, inputs...)
		api.batches = append(api.batches, inputs)
		if api.hook != nil {
			api.hook()
		}
		if api.status != 0 {
			w.WriteHeader(api.status)
			fmt.Fprint(w, "secret provider response")
			return
		}
		if api.failOnRequest == len(api.batches) {
			w.WriteHeader(429)
			return
		}
		if api.response != "" {
			fmt.Fprint(w, api.response)
			return
		}
		var data []any
		// Deliberately return reverse order to exercise response-index mapping.
		for i := len(inputs) - 1; i >= 0; i-- {
			vector := []float32{0, 1}
			if strings.Contains(inputs[i], "fruit") || strings.Contains(inputs[i], "apple") {
				vector = []float32{1, 0}
			}
			data = append(data, map[string]any{"index": i, "embedding": vector})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	t.Cleanup(server.Close)
	t.Setenv("MEANS_EMBEDDING_BASE_URL", server.URL)
	t.Setenv("MEANS_EMBEDDING_API_KEY", "test-key")
	t.Setenv("OPENAI_API_KEY", "")
	return api
}

func embeddingCollection(t *testing.T) *Collection {
	t.Helper()
	c, err := InitConfig(t.TempDir(), []byte("---\nschema: true\nembeddings:\n  provider: openai\n  dimensions: 2\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func embeddingRecords(t *testing.T, c *Collection) map[string]embeddingRecord {
	t.Helper()
	s, err := c.openEmbeddings()
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	return s.records
}

func TestEmbeddingLifecycle(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	doc, err := c.Create(map[string]any{"title": "apple", "data": map[string]any{"status": "todo"}}, "fruit body")
	if err != nil {
		t.Fatal(err)
	}
	if len(api.inputs) != 1 || api.inputs[0] != "Title: apple\n\nfruit body" {
		t.Fatalf("inputs: %v", api.inputs)
	}
	if len(embeddingRecords(t, c)) != 1 {
		t.Fatal("create did not persist embedding")
	}
	doc, err = c.Update(doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"data": map[string]any{"status": "done"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(api.inputs) != 1 {
		t.Fatal("data-only update made an API call")
	}
	doc, err = c.Update(doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"title": "engine"}})
	if err != nil {
		t.Fatal(err)
	}
	body := "machine body"
	doc, err = c.Update(doc.ID, doc.Hash, Patch{Body: &body})
	if err != nil {
		t.Fatal(err)
	}
	if len(api.inputs) != 3 {
		t.Fatalf("title/body update calls: %v", api.inputs)
	}
	api.status = 503
	if _, err := c.Delete(doc.ID, doc.Hash); err != nil {
		t.Fatal(err)
	}
	if len(api.inputs) != 3 || len(embeddingRecords(t, c)) != 0 {
		t.Fatal("delete called API or retained vector")
	}
}

func TestEmbeddingDisabledIsOffline(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	for _, setting := range []string{"", "embeddings: false\n"} {
		c, err := InitConfig(t.TempDir(), []byte("---\nschema: true\n"+setting+"---\n"))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := c.Create(map[string]any{"title": "offline"}, "")
		if err != nil {
			t.Fatal(err)
		}
		body := "changed"
		doc, err = c.Update(doc.ID, doc.Hash, Patch{Body: &body})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Delete(doc.ID, doc.Hash); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(c.Path, ".means")); !os.IsNotExist(err) {
			t.Fatalf("unexpected cache: %v", err)
		}
		if _, err := c.Search(SearchOptions{Semantic: true, Query: "anything"}); err == nil {
			t.Fatal("disabled semantic search accepted")
		}
	}
	if len(api.inputs) != 0 {
		t.Fatal("disabled collection called API")
	}
}

func TestEmbeddingFailuresPreserveDocuments(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	doc, err := c.Create(map[string]any{"title": "original"}, "original")
	if err != nil {
		t.Fatal(err)
	}
	api.status = 429
	body := "changed"
	if _, err := c.Update(doc.ID, doc.Hash, Patch{Body: &body}); err == nil || !strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("update error: %v", err)
	}
	after, err := c.Read(doc.ID)
	if err != nil || after.Hash != doc.Hash {
		t.Fatalf("failed update changed document: %v", err)
	}
	if _, err := c.Create(map[string]any{"title": "failed"}, ""); err == nil {
		t.Fatal("failed create succeeded")
	}
	listed, err := c.List(0)
	if err != nil || len(listed.Documents) != 1 {
		t.Fatalf("failed create left document: %+v %v", listed, err)
	}
	api.status = 0
	if err := os.Remove(filepath.Join(c.Path, ".means", "embeddings.jsonl")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(c.Path, ".means", "embeddings.jsonl"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Update(doc.ID, doc.Hash, Patch{Body: &body}); err == nil {
		t.Fatal("cache failure ignored")
	}
	if _, err := c.Delete(doc.ID, doc.Hash); err == nil {
		t.Fatal("delete cache failure ignored")
	}
	after, err = c.Read(doc.ID)
	if err != nil || after.Hash != doc.Hash {
		t.Fatal("cache failure changed document")
	}
}

func TestEmbeddingUpdateRechecksAfterAPI(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	doc, err := c.Create(map[string]any{"title": "original"}, "original")
	if err != nil {
		t.Fatal(err)
	}
	api.hook = func() {
		if err := os.WriteFile(filepath.Join(c.Path, doc.filename), []byte(doc.Markdown+"external"), 0o644); err != nil {
			t.Error(err)
		}
	}
	body := "proposed"
	if _, err := c.Update(doc.ID, doc.Hash, Patch{Body: &body}); err == nil || !strings.Contains(err.Error(), "content has changed") {
		t.Fatalf("want conflict, got %v", err)
	}
	api.hook = nil
	current, err := c.Read(doc.ID)
	if err != nil || current.Body != "originalexternal" {
		t.Fatalf("lost external edit: %+v %v", current, err)
	}
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	if embeddingRecords(t, c)[doc.ID].Hash != contentHash(embeddingInput(current.Frontmatter, current.Body)) {
		t.Fatal("failed mutation left usable stale vector")
	}
}

func TestSemanticSearchUsesStoredVectorsAndFilters(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	fruit, err := c.Create(map[string]any{"title": "apple", "data": map[string]any{"count": json.Number("9007199254740993")}}, "orchard")
	if err != nil {
		t.Fatal(err)
	}
	other, err := c.Create(map[string]any{"title": "engine"}, "machine")
	if err != nil {
		t.Fatal(err)
	}
	allScores := -1.0
	result, err := c.Search(SearchOptions{Query: "fruit", Semantic: true, Limit: 1, WithBody: true, MinScore: &allScores})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Documents) != 1 || result.Documents[0].ID != fruit.ID || !result.HasMore || *result.Documents[0].Score != 1 || result.Documents[0].Match != nil || *result.Documents[0].Body != "orchard" {
		t.Fatalf("search: %+v", result)
	}
	result, err = c.Search(SearchOptions{Query: "machine", Semantic: true, Limit: 1, Filter: map[string]any{"count": json.Number("9007199254740993")}, MinScore: &allScores})
	if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != fruit.ID || result.HasMore {
		t.Fatalf("filter before limit: %+v %v", result, err)
	}
	if err := os.WriteFile(filepath.Join(c.Path, other.filename), []byte(strings.Replace(other.Markdown, "machine", "fruit", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(c.Path, fruit.filename)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Path, "bad.md"), []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := len(api.inputs)
	result, err = c.Search(SearchOptions{Query: "fruit", Semantic: true, Limit: 1})
	if err != nil || len(result.Documents) != 0 || len(result.Diagnostics) != 1 {
		t.Fatalf("stale vectors: %+v %v", result, err)
	}
	if len(api.inputs) != before || len(embeddingRecords(t, c)) != 2 {
		t.Fatal("search generated vectors or pruned deletion")
	}
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	result, err = c.Search(SearchOptions{Query: "fruit", Semantic: true})
	if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != other.ID || *result.Documents[0].Score != 1 {
		t.Fatalf("explicit refresh: %+v %v", result, err)
	}
	if err := os.WriteFile(filepath.Join(c.Path, "duplicate.md"), []byte(other.Markdown), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err = c.Search(SearchOptions{Query: "fruit", Semantic: true})
	if err != nil || len(result.Documents) != 0 || len(result.Diagnostics) != 3 || len(embeddingRecords(t, c)) != 1 {
		t.Fatalf("duplicates: %+v %v", result, err)
	}
}

func TestSemanticSearchNeverRefreshesCache(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	doc, err := c.Create(map[string]any{"title": "apple", "data": map[string]any{"wanted": true}}, "fruit")
	if err != nil {
		t.Fatal(err)
	}
	// A directly imported document has no vector and must never be sent to the API.
	if err := os.WriteFile(filepath.Join(c.Path, "import.md"), []byte("---\nid: abcde\ntitle: imported\ndata: {}\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(c.Path, ".means", "embeddings.jsonl")
	for _, filter := range []map[string]any{nil, {"wanted": true}} {
		before, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		calls := len(api.inputs)
		result, err := c.Search(SearchOptions{Query: "fruit", Semantic: true, Filter: filter})
		if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != doc.ID {
			t.Fatalf("search: %+v %v", result, err)
		}
		after, err := os.ReadFile(filename)
		if err != nil || string(before) != string(after) || len(api.inputs) != calls+1 || api.inputs[calls] != "fruit" {
			t.Fatal("search must only embed the query and leave cache bytes unchanged")
		}
	}
	for _, state := range []string{"missing", "corrupt", "profile mismatch"} {
		t.Run(state, func(t *testing.T) {
			switch state {
			case "missing":
				if err := os.Remove(filename); err != nil {
					t.Fatal(err)
				}
			case "corrupt":
				if err := os.WriteFile(filename, []byte("broken"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "profile mismatch":
				if _, err := c.Embed(); err != nil {
					t.Fatal(err)
				}
				t.Setenv("MEANS_EMBEDDING_BASE_URL", "http://127.0.0.1:1")
			}
			before, beforeErr := os.ReadFile(filename)
			calls := len(api.inputs)
			result, err := c.Search(SearchOptions{Query: "fruit", Semantic: true})
			if err != nil || len(result.Documents) != 0 || len(api.inputs) != calls {
				t.Fatalf("unavailable vectors: %+v %v", result, err)
			}
			after, afterErr := os.ReadFile(filename)
			if string(before) != string(after) || (beforeErr == nil) != (afterErr == nil) {
				t.Fatal("search changed the cache")
			}
		})
	}
}

func TestSemanticQueryDoesNotHoldEmbeddingLock(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	doc, err := c.Create(map[string]any{"title": "apple"}, "fruit")
	if err != nil {
		t.Fatal(err)
	}
	// Run deletion while the query request is in flight. It needs the cache
	// lock but no API call, and must complete before this response is sent.
	api.hook = func() {
		if _, err := c.Delete(doc.ID, doc.Hash); err != nil {
			t.Errorf("delete during query request: %v", err)
		}
	}
	if _, err := c.Search(SearchOptions{Query: "fruit", Semantic: true}); err != nil {
		t.Fatal(err)
	}
	if len(embeddingRecords(t, c)) != 0 {
		t.Fatal("search overwrote the deletion with its snapshot")
	}
}

func TestHeldEmbeddingLockBlocksWritesButNotSemanticSearch(t *testing.T) {
	fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	if _, err := c.Create(map[string]any{"title": "apple"}, "fruit"); err != nil {
		t.Fatal(err)
	}
	// Simulate a long-running embed or mutation in another process.
	lock := flock.New(filepath.Join(c.Path, ".means", "embeddings.lock"))
	if err := lock.Lock(); err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()
	result, err := c.Search(SearchOptions{Query: "fruit", Semantic: true})
	if err != nil || len(result.Documents) != 1 {
		t.Fatalf("search during held lock: %+v %v", result, err)
	}
	if _, err := c.Create(map[string]any{"title": "blocked"}, ""); err == nil || err.Error() != "embeddings are busy; retry after the current operation finishes" {
		t.Fatalf("create during held lock: %v", err)
	}
}

func TestEmbeddingCacheRecoveryAndBoundaries(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	doc, err := c.Create(map[string]any{"title": "apple"}, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{"truncated", "{\"version\":999}\n"} {
		if err := os.WriteFile(filepath.Join(c.Path, ".means", "embeddings.jsonl"), []byte(invalid), 0o600); err != nil {
			t.Fatal(err)
		}
		before := len(api.inputs)
		if _, err := c.Embed(); err != nil {
			t.Fatal(err)
		}
		if len(api.inputs) != before+1 {
			t.Fatal("cache not rebuilt")
		}
	}
	nested := filepath.Join(c.Path, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(c.Path, doc.filename), filepath.Join(nested, doc.filename)); err != nil {
		t.Fatal(err)
	}
	before := len(api.inputs)
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	if len(api.inputs) != before {
		t.Fatal("move regenerated embedding")
	}
	if err := os.WriteFile(filepath.Join(nested, ConfigFilename), []byte("invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	if len(embeddingRecords(t, c)) != 0 {
		t.Fatal("nested collection retained")
	}
}

func TestEmbeddingResponseValidation(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	for _, response := range []string{`{`, `{"data":[]}`, `{"data":[{"index":0,"embedding":[1]}]}`, `{"data":[{"index":1,"embedding":[1,0]}]}`, `{"data":[{"index":0,"embedding":[0,0]}]}`, `{"data":[{"index":0,"embedding":[1e100,0]}]}`} {
		api.response = response
		if _, err := c.Create(map[string]any{"title": "bad response"}, ""); err == nil {
			t.Fatalf("accepted %s", response)
		}
	}
	listed, err := c.List(0)
	if err != nil || len(listed.Documents) != 0 {
		t.Fatal("invalid responses created documents")
	}
}

func TestEmbeddingConfigValidation(t *testing.T) {
	for _, value := range []string{"true", "null", "[]", "{}", "{provider: wrong}", "{provider: true}", "{provider: openai, model: ''}", "{provider: openai, model: openai/text-embedding-3-small}", "{provider: openrouter, model: text-embedding-3-small}", "{provider: openai, model: wrong}", "{provider: openai, dimensions: 0}", "{provider: openai, dimensions: 1.5}", "{provider: openai, dimensions: 1537}", "{provider: openai, dimensions: '2'}", "{provider: openai, api_key: secret}"} {
		if _, err := InitConfig(t.TempDir(), []byte("---\nschema: true\nembeddings: "+value+"\n---\n")); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	for _, model := range []string{"text-embedding-3-small", "text-embedding-3-large", "openai/text-embedding-3-small"} {
		provider := "openai"
		if strings.HasPrefix(model, "openai/") {
			provider = "openrouter"
		}
		c, err := InitConfig(t.TempDir(), []byte("---\nschema: true\nembeddings: {provider: "+provider+", model: "+model+"}\n---\n"))
		if err != nil || c.Embeddings().Model != model {
			t.Fatalf("model %s: %v", model, err)
		}
	}
}

func TestEmbeddingConcurrentCreates(t *testing.T) {
	fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Go(func() {
			if _, err := c.Create(map[string]any{"title": "parallel"}, ""); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if len(embeddingRecords(t, c)) != 4 {
		t.Fatal("concurrent create lost vectors")
	}
}

func TestEmbeddingNoOpRechecksAfterAPI(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	doc, err := c.Create(map[string]any{"title": "original"}, "original")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(c.Path, ".means", "embeddings.jsonl")); err != nil {
		t.Fatal(err)
	}
	api.hook = func() {
		if err := os.WriteFile(filepath.Join(c.Path, doc.filename), []byte(doc.Markdown+"external"), 0o644); err != nil {
			t.Error(err)
		}
	}
	if _, err := c.Update(doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{}}); err == nil || !strings.Contains(err.Error(), "content has changed") {
		t.Fatalf("no-op missed conflict: %v", err)
	}
}

func TestEmbeddingMissingKeyAndCancellation(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	t.Setenv("MEANS_EMBEDDING_API_KEY", "")
	if _, err := c.Create(map[string]any{"title": "no key"}, ""); err == nil {
		t.Fatal("missing key accepted")
	}
	t.Setenv("MEANS_EMBEDDING_API_KEY", "test-key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.WithContext(ctx).Create(map[string]any{"title": "canceled"}, ""); err == nil {
		t.Fatal("cancellation ignored")
	}
	if len(api.inputs) != 0 {
		t.Fatal("sent a canceled request")
	}
	listed, err := c.List(0)
	if err != nil || len(listed.Documents) != 0 {
		t.Fatal("failed request left document")
	}
}

func TestEmbeddingCacheSymlinksFail(t *testing.T) {
	fakeEmbeddingAPI(t)
	for _, name := range []string{".means", ".means/embeddings.lock", ".means/embeddings.jsonl"} {
		t.Run(name, func(t *testing.T) {
			c := embeddingCollection(t)
			target := filepath.Join(t.TempDir(), "target")
			if err := os.WriteFile(target, []byte("untouched"), 0o600); err != nil {
				t.Fatal(err)
			}
			if name != ".means" {
				if err := os.Mkdir(filepath.Join(c.Path, ".means"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(target, filepath.Join(c.Path, name)); err != nil {
				t.Fatal(err)
			}
			if _, err := c.Create(map[string]any{"title": "blocked"}, ""); err == nil {
				t.Fatal("symlink accepted")
			}
			data, err := os.ReadFile(target)
			if err != nil || string(data) != "untouched" {
				t.Fatal("symlink target changed")
			}
		})
	}
}

func TestEmbeddingProfileChangesInvalidateCache(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	if _, err := c.Create(map[string]any{"title": "original"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := SetSchema(c.Path, []byte(`{"type":"object"}`), false); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(c.Path)
	if err != nil || loaded.Embeddings() == nil || loaded.Embeddings().Dimensions != 2 {
		t.Fatalf("schema set lost config: %v", err)
	}
	// A profile mismatch must not reuse vectors even if the document input is unchanged.
	s, err := c.openEmbeddings()
	if err != nil {
		t.Fatal(err)
	}
	s.profile = "obsolete-profile"
	s.dirty = true
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	s.close()
	before := len(api.inputs)
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	if len(api.inputs) != before+1 {
		t.Fatal("profile mismatch reused a vector")
	}
}
