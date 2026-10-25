package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hmans/means/internal/collection"
)

func TestSemanticThresholdCLI(t *testing.T) {
	var query atomic.Bool
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if query.Load() {
			w.Write([]byte(`{"data":[{"index":0,"embedding":[0,1]}]}`))
		} else {
			w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0]}]}`))
		}
	}))
	defer api.Close()
	t.Setenv("MEANS_EMBEDDING_BASE_URL", api.URL)
	t.Setenv("MEANS_EMBEDDING_API_KEY", "test")
	c, err := collection.InitConfig(t.TempDir(), []byte("---\nschema: true\nembeddings: {provider: openai, dimensions: 2}\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Create(map[string]any{"title": "document"}, ""); err != nil {
		t.Fatal(err)
	}
	query.Store(true)
	for _, tt := range []struct {
		args  []string
		count int
	}{
		{nil, 0}, {[]string{"--min-score", "0"}, 1}, {[]string{"--min-score", "-1"}, 1},
	} {
		var out, stderr bytes.Buffer
		args := append([]string{"search", "query", "--semantic", "--path", c.Path, "--json"}, tt.args...)
		if err := Execute(args, nil, &out, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		var result collection.SearchResult
		if err := json.Unmarshal(out.Bytes(), &result); err != nil || len(result.Documents) != tt.count {
			t.Fatalf("result: %s %v", out.String(), err)
		}
	}
	var out, stderr bytes.Buffer
	if err := Execute([]string{"search", "query", "--semantic", "--min-score", "0", "--path", c.Path}, nil, &out, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "SCORE") || !strings.Contains(out.String(), "0.000") || !strings.Contains(out.String(), "document") {
		t.Fatal(out.String())
	}
	for _, args := range [][]string{
		{"--min-score", "0"}, {"--semantic", "--min-score", "NaN"}, {"--semantic", "--min-score", "+Inf"}, {"--semantic", "--min-score", "1.1"},
	} {
		out.Reset()
		stderr.Reset()
		if err := Execute(append([]string{"search", "query", "--path", c.Path, "--json"}, args...), nil, &out, &stderr, "test"); err == nil || !json.Valid(stderr.Bytes()) {
			t.Fatalf("accepted invalid args: %v %s", args, stderr.String())
		}
	}
}
