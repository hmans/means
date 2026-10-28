package mcpserver

import (
	"context"
	"github.com/hmans/means/internal/collection"
	"github.com/hmans/means/internal/userconfig"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Never inherit a developer's embedding settings or modify their user config.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "means-mcpserver-tests-")
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

func TestMCPBootstrapsAndSnapshotsUserConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	server, err := New(c.Path, "test")
	if err != nil {
		t.Fatal(err)
	}
	filename, _ := userconfig.Path()
	data, err := os.ReadFile(filename)
	if err != nil || string(data) != userconfig.Example {
		t.Fatal("MCP did not bootstrap config")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connect := func(server *mcp.Server) *mcp.ClientSession {
		a, b := mcp.NewInMemoryTransports()
		ss, err := server.Connect(ctx, b, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { ss.Close() })
		client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
		session, err := client.Connect(ctx, a, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { session.Close() })
		return session
	}
	session := connect(server)
	if info := callTool(t, ctx, session, "info", `{}`); info["embeddings"] != nil {
		t.Fatal("starter enabled embeddings")
	}
	if err := os.WriteFile(filename, []byte("embeddings: {provider: openrouter}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if info := callTool(t, ctx, session, "info", `{}`); info["embeddings"] != nil {
		t.Fatal("running MCP changed host settings")
	}
	restarted, err := New(c.Path, "test")
	if err != nil {
		t.Fatal(err)
	}
	info := callTool(t, ctx, connect(restarted), "info", `{}`)
	cfg, ok := info["embeddings"].(map[string]any)
	if !ok || cfg["provider"] != "openrouter" {
		t.Fatalf("restart didn't load config: %v", info)
	}
}

func TestMCPBootstrapFailureAllowsOfflineStartup(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	if err := os.WriteFile(filepath.Join(base, "means"), []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var messages []string
	if _, err := NewWithEmbeddingProgress(c.Path, "test", func(s string) { messages = append(messages, s) }); err != nil || len(messages) != 1 {
		t.Fatalf("startup: %v %v", messages, err)
	}
	messages = nil
	if _, err := NewWithEmbeddingProgress(filepath.Join(c.Path, "missing"), "test", func(s string) { messages = append(messages, s) }); err == nil || len(messages) != 0 {
		t.Fatalf("failed startup mixed warning and error: %v %v", messages, err)
	}
}

func TestDotenvStartup(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	const key = "MEANS_TEST_MCP_DOTENV"
	t.Setenv(key, "")
	os.Unsetenv(key)
	if err := os.WriteFile(filepath.Join(c.Path, ".env"), []byte(key+"=loaded\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(c.Path, "test"); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(key) != "loaded" {
		t.Fatal("MCP did not load collection .env")
	}
	if err := os.WriteFile(filepath.Join(c.Path, ".env"), []byte("BROKEN='secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(c.Path, "test"); err == nil {
		t.Fatal("invalid dotenv accepted")
	}
}
