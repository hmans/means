package collection_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hmans/means/collection"
)

func TestPublicWorkflow(t *testing.T) {
	// Host files must not affect library operations or process environment.
	host := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", host)
	dir := t.TempDir()
	const key = "MEANS_PUBLIC_API_TEST_ENV"
	t.Setenv(key, "host")
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(key+"=collection\nBROKEN='"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := collection.Init(dir, []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	input := map[string]any{"count": json.Number("9007199254740993"), "status": "draft"}
	doc, err := c.Create(ctx, collection.CreateOptions{Title: "A note", Data: input, Body: "Exact body\n"})
	if err != nil {
		t.Fatal(err)
	}
	input["count"] = 0
	current, err := c.Read(ctx, doc.ID)
	if err != nil || current.Hash != doc.Hash || current.Body != "Exact body\n" || current.Frontmatter["data"].(map[string]any)["count"] != json.Number("9007199254740993") {
		t.Fatalf("read = %+v, %v", current, err)
	}
	listed, err := c.List(ctx, collection.ListOptions{Filter: map[string]any{"status": "draft"}})
	if err != nil || len(listed.Documents) != 1 || listed.HasMore || len(listed.Diagnostics) != 0 {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	found, err := c.Search(ctx, collection.SearchOptions{Query: "note", Filter: map[string]any{"status": "draft"}})
	if err != nil || len(found.Documents) != 1 || found.Documents[0].ID != doc.ID {
		t.Fatalf("search = %+v, %v", found, err)
	}
	body := "Changed body"
	updated, err := c.Update(ctx, doc.ID, current.Hash, collection.Patch{Body: &body})
	if err != nil || updated.Hash == doc.Hash {
		t.Fatalf("update = %+v, %v", updated, err)
	}
	if _, err := c.Update(ctx, doc.ID, doc.Hash, collection.Patch{Body: &body}); !errors.Is(err, collection.ErrConflict) {
		t.Fatalf("stale update = %v", err)
	}
	if _, err := c.Delete(ctx, doc.ID, doc.Hash); !errors.Is(err, collection.ErrConflict) {
		t.Fatalf("stale delete = %v", err)
	}
	if _, err := c.Delete(ctx, doc.ID, updated.Hash); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Read(ctx, doc.ID); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing read = %v", err)
	}
	if _, err := os.Stat(filepath.Join(host, "means")); !errors.Is(err, fs.ErrNotExist) || os.Getenv(key) != "host" {
		t.Fatal("library loaded or bootstrapped host configuration")
	}
}

func TestPublicConfigurationSnapshot(t *testing.T) {
	ctx := context.Background()
	c, err := collection.InitConfig(t.TempDir(), []byte("---\ntitle: Notes\nschema: true\napp:\n  views:\n    - title: Drafts\n      columns: [title]\n---\nOriginal guidance\n"))
	if err != nil {
		t.Fatal(err)
	}
	schema := c.Schema()
	schema[0] = 'x'
	app := c.App()
	app.Views[0].Title = "Changed"
	app.Views[0].Columns[0] = "id"
	if string(c.Schema()) != "true" || c.App().Views[0].Title != "Drafts" || c.App().Views[0].Columns[0] != "title" {
		t.Fatal("accessors exposed mutable snapshot state")
	}
	doc, err := c.Create(ctx, collection.CreateOptions{Title: "Keep"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := collection.SetSchema(ctx, c.Path(), []byte(`false`), false)
	if !errors.Is(err, collection.ErrSchemaIncompatible) || result == nil || result.Installed || result.Healthy || len(result.Diagnostics) != 1 {
		t.Fatalf("rejected schema = %+v, %v", result, err)
	}
	if _, err := collection.SetSchema(ctx, c.Path(), []byte(`false`), true); err != nil {
		t.Fatal(err)
	}
	loaded, err := collection.Load(c.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateData(map[string]any{}); err != nil {
		t.Fatalf("snapshot changed: %v", err)
	}
	if err := loaded.ValidateData(map[string]any{}); !errors.Is(err, collection.ErrValidation) {
		t.Fatalf("reload missed schema change: %v", err)
	}
	if _, err := loaded.Read(ctx, doc.ID); err != nil {
		t.Fatalf("schema-invalid document must remain readable: %v", err)
	}
	if _, err := loaded.SetInstructions(ctx, c.Describe().Hash, "Stale"); !errors.Is(err, collection.ErrConflict) {
		t.Fatalf("stale configuration hash = %v", err)
	}
	if _, err := loaded.SetInstructions(ctx, loaded.Describe().Hash, "New guidance\n"); err != nil {
		t.Fatal(err)
	}
	if loaded.Describe().Instructions != "Original guidance\n" {
		t.Fatal("instruction update mutated the loaded snapshot")
	}
}

func TestPublicOperationCancellation(t *testing.T) {
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(context.Background(), collection.CreateOptions{Title: "Keep"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadDir(c.Path())
	if err != nil {
		t.Fatal(err)
	}
	ops := map[string]func(context.Context) error{
		"create": func(ctx context.Context) error {
			_, err := c.Create(ctx, collection.CreateOptions{Title: "Canceled"})
			return err
		},
		"read": func(ctx context.Context) error { _, err := c.Read(ctx, doc.ID); return err },
		"update": func(ctx context.Context) error {
			_, err := c.Update(ctx, doc.ID, doc.Hash, collection.Patch{})
			return err
		},
		"replace": func(ctx context.Context) error { _, err := c.ReplaceMarkdown(ctx, doc.ID, doc.Hash, nil); return err },
		"delete":  func(ctx context.Context) error { _, err := c.Delete(ctx, doc.ID, doc.Hash); return err },
		"list":    func(ctx context.Context) error { _, err := c.List(ctx, collection.ListOptions{}); return err },
		"search": func(ctx context.Context) error {
			_, err := c.Search(ctx, collection.SearchOptions{Query: "Keep"})
			return err
		},
		"doctor": func(ctx context.Context) error { _, err := c.Doctor(ctx); return err },
		"embed": func(ctx context.Context) error {
			_, err := c.Embed(ctx, collection.EmbedOptions{Force: true})
			return err
		},
		"schema": func(ctx context.Context) error {
			_, err := collection.SetSchema(ctx, c.Path(), []byte(`false`), true)
			return err
		},
		"instructions": func(ctx context.Context) error {
			_, err := c.SetInstructions(ctx, c.Describe().Hash, "Canceled")
			return err
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			if err := op(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled operation = %v", err)
			}
			if err := op(nil); err == nil {
				t.Fatal("accepted nil context")
			}
		})
	}
	after, err := os.ReadDir(c.Path())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("canceled operation changed files: %v", err)
	}
	current, err := c.Read(context.Background(), doc.ID)
	if err != nil || current.Hash != doc.Hash || c.Describe().Instructions != "# Collection guidance\n\nDescribe this collection and its workflow here.\n" {
		t.Fatalf("cancellation leaked into another operation: %+v, %v", current, err)
	}
}

func TestPublicCreationAndDiagnostics(t *testing.T) {
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := c.Create(ctx, collection.CreateOptions{}); !errors.Is(err, collection.ErrValidation) {
		t.Fatalf("invalid title = %v", err)
	}
	if _, err := c.Create(ctx, collection.CreateOptions{Title: "Invalid data", Data: map[string]any{"unknown": true}}); !errors.Is(err, collection.ErrValidation) {
		t.Fatalf("invalid data = %v", err)
	}
	doc, err := c.Create(ctx, collection.CreateOptions{Title: "Keep"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Path(), "duplicate.md"), []byte(doc.Markdown), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Read(ctx, doc.ID); !errors.Is(err, collection.ErrDuplicateID) {
		t.Fatalf("duplicate read = %v", err)
	}
	if err := os.WriteFile(filepath.Join(c.Path(), "broken.md"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	listed, err := c.List(ctx, collection.ListOptions{Limit: 1})
	if err != nil || len(listed.Documents) != 0 || len(listed.Diagnostics) != 3 {
		t.Fatalf("document diagnostics = %+v, %v", listed, err)
	}
}

func TestPublicNativeGoValues(t *testing.T) {
	c, err := collection.Init(t.TempDir(), []byte(`{"type":"object","properties":{"tags":{"type":"array","items":{"type":"string"}},"count":{"type":"integer"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	data := map[string]any{"tags": []string{"go"}, "count": 42}
	doc, err := c.Create(ctx, collection.CreateOptions{Title: "Go values", Data: data})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("validate", func(t *testing.T) {
		if err := c.ValidateData(data); err != nil {
			t.Fatalf("validation rejected Create input: %v", err)
		}
		if err := c.Validate(map[string]any{"id": doc.ID, "title": "Go values", "data": data}); err != nil {
			t.Fatalf("envelope validation rejected Create input: %v", err)
		}
	})
	t.Run("list", func(t *testing.T) {
		listed, err := c.List(ctx, collection.ListOptions{Filter: data})
		if err != nil || len(listed.Documents) != 1 {
			t.Fatalf("native filter missed document: %+v, %v", listed, err)
		}
	})
	for _, query := range []collection.SearchOptions{{}, {Query: "Go"}, {Query: "Go", Literal: true}} {
		t.Run("search", func(t *testing.T) {
			query.Filter = data
			found, err := c.Search(ctx, query)
			if err != nil || len(found.Documents) != 1 {
				t.Fatalf("native filter missed document: %+v, %v", found, err)
			}
		})
	}
	if _, ok := data["tags"].([]string); !ok || data["count"] != 42 {
		t.Fatal("operation changed caller values")
	}
}
