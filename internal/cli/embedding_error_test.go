package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hmans/means/internal/collection"
)

func TestEmbeddingFailureKeepsJSONErrorParseable(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer api.Close()
	t.Setenv("MEANS_EMBEDDING_BASE_URL", api.URL)
	t.Setenv("MEANS_EMBEDDING_API_KEY", "test")
	c, err := collection.InitConfig(t.TempDir(), []byte("---\nschema: true\nembeddings: {provider: openai}\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	err = Execute([]string{"create", "--path", c.Path, "--input", "-", "--json"}, strings.NewReader(`{"frontmatter":{"title":"failure"}}`), &out, &stderr, "test")
	if err == nil || out.Len() != 0 || !json.Valid(stderr.Bytes()) || !strings.Contains(stderr.String(), "503") {
		t.Fatalf("error stream: %v %s", err, stderr.String())
	}
	listed, err := c.List(0)
	if err != nil || len(listed.Documents) != 0 {
		t.Fatal("failed embedding wrote document")
	}
}
