package cli

import (
	"bytes"
	"encoding/json"
	"github.com/hmans/means/internal/collection"
	"github.com/hmans/means/internal/userconfig"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// Never inherit a developer's embedding settings or modify their user config.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "means-cli-tests-")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_CONFIG_HOME", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func embeddingTestCollection(t *testing.T, path, settings string) (*collection.Collection, error) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	filename := filepath.Join(dir, "means", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		return nil, err
	}
	data := []byte("embeddings: " + settings + "\n")
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return nil, err
	}
	cfg, err := userconfig.Parse(data)
	if err != nil {
		return nil, err
	}
	c, err := collection.Init(path, []byte(`true`))
	if err != nil {
		return nil, err
	}
	return c.WithEmbeddings(cfg.Embeddings), nil
}

func TestUserConfigBootstrapAndFailures(t *testing.T) {
	for _, command := range [][]string{{"--help"}, {"version"}, {"agents"}, {"init", "--path", t.TempDir()}} {
		t.Run(command[0], func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)
			var out, stderr bytes.Buffer
			if err := Execute(command, nil, &out, &stderr, "test"); err != nil || stderr.Len() != 0 {
				t.Fatalf("execute: %v %s", err, stderr.String())
			}
			data, err := os.ReadFile(filepath.Join(dir, "means", "config.yaml"))
			if err != nil || string(data) != userconfig.Example {
				t.Fatal("missing starter config")
			}
		})
	}
	t.Run("unwritable", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)
		if err := os.WriteFile(filepath.Join(dir, "means"), []byte("block"), 0600); err != nil {
			t.Fatal(err)
		}
		c, err := collection.Init(t.TempDir(), nil)
		if err != nil {
			t.Fatal(err)
		}
		var out, stderr bytes.Buffer
		if err := Execute([]string{"list", "--path", c.Path, "--json"}, nil, &out, &stderr, "test"); err != nil || !json.Valid(out.Bytes()) || !json.Valid(stderr.Bytes()) || !strings.Contains(stderr.String(), "warning") {
			t.Fatalf("offline failure: %v %s %s", err, out.String(), stderr.String())
		}
		out.Reset()
		stderr.Reset()
		if err := Execute([]string{"embed", "--path", c.Path, "--json"}, nil, &out, &stderr, "test"); err == nil || !json.Valid(stderr.Bytes()) || strings.Contains(stderr.String(), `"warning"`) {
			t.Fatalf("error framing: %v %s", err, stderr.String())
		}
	})
	t.Run("malformed", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)
		if err := os.MkdirAll(filepath.Join(dir, "means"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "means", "config.yaml"), []byte("embeddings: true"), 0600); err != nil {
			t.Fatal(err)
		}
		var out, stderr bytes.Buffer
		if err := Execute([]string{"list", "--json"}, nil, &out, &stderr, "test"); err == nil || !json.Valid(stderr.Bytes()) || out.Len() != 0 {
			t.Fatalf("invalid config: %v %s", err, stderr.String())
		}
		out.Reset()
		stderr.Reset()
		if err := Execute([]string{"embed", "--help", "--json"}, nil, &out, &stderr, "test"); err != nil || !json.Valid(out.Bytes()) || stderr.Len() != 0 {
			t.Fatalf("help inaccessible: %v %s", err, stderr.String())
		}
	})
}

func TestUserConfigAppliesAcrossCollections(t *testing.T) {
	var calls atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0]}]}`))
	}))
	defer api.Close()
	t.Setenv("MEANS_EMBEDDING_BASE_URL", api.URL)
	t.Setenv("MEANS_EMBEDDING_API_KEY", "test")
	first, err := embeddingTestCollection(t, t.TempDir(), "{provider: openai, dimensions: 2}")
	if err != nil {
		t.Fatal(err)
	}
	second, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []*collection.Collection{first, second} {
		var out, stderr bytes.Buffer
		err := Execute([]string{"create", "--path", c.Path, "--input", "-", "--json"}, strings.NewReader(`{"frontmatter":{"title":"sample"}}`), &out, &stderr, "test")
		if err != nil || !json.Valid(out.Bytes()) || stderr.Len() != 0 {
			t.Fatalf("create: %v %s", err, stderr.String())
		}
		data, err := os.ReadFile(filepath.Join(c.Path, "MEANS.md"))
		if err != nil || strings.Contains(string(data), "embeddings") {
			t.Fatal("host config leaked into collection")
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("calls: %d", calls.Load())
	}
	filename, _ := userconfig.Path()
	if err := os.WriteFile(filename, []byte("embeddings: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if err := Execute([]string{"create", "--path", first.Path, "--input", "-", "--json"}, strings.NewReader(`{"frontmatter":{"title":"offline"}}`), &out, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatal("CLI did not reload disabled config")
	}
}
