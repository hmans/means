package collection_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hmans/means/collection"
)

func TestPublicEmbeddingSettings(t *testing.T) {
	var requests atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var input struct {
			Model      string `json:"model"`
			Dimensions int    `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Model != "text-embedding-3-small" || input.Dimensions != 512 {
			t.Errorf("embedding defaults: %+v, %v", input, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		vector := make([]float32, 512)
		vector[0] = 1
		json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"index": 0, "embedding": vector}}})
	}))
	defer api.Close()
	t.Setenv("MEANS_EMBEDDING_BASE_URL", api.URL)
	t.Setenv("MEANS_EMBEDDING_API_KEY", "test-key")
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	settings := collection.EmbeddingConfig{Provider: "openai"}
	enabled := c.WithEmbeddings(&settings)
	settings.Provider = "changed"
	copy := enabled.Embeddings()
	copy.Provider = "also changed"
	ctx := context.Background()
	if _, err := enabled.Create(ctx, collection.CreateOptions{Title: "Embedded"}); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 || c.Embeddings() != nil {
		t.Fatal("settings were not copied or defaults required host configuration")
	}
	if _, err := c.WithEmbeddings(&settings).Create(ctx, collection.CreateOptions{Title: "Invalid settings"}); err == nil {
		t.Fatal("invalid public embedding settings accepted")
	}
	listed, err := c.List(ctx, collection.ListOptions{})
	if err != nil || len(listed.Documents) != 1 || requests.Load() != 1 {
		t.Fatalf("invalid embedding settings wrote a document: %+v, %v", listed, err)
	}
	resolved, err := (collection.EmbeddingConfig{Provider: "openrouter"}).WithDefaults()
	if err != nil || resolved.Model != "openai/text-embedding-3-small" || resolved.Dimensions != 512 {
		t.Fatalf("OpenRouter defaults: %+v, %v", resolved, err)
	}
}
