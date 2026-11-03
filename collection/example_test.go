package collection_test

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/hmans/means/collection"
)

func ExampleCollection_Create() {
	dir, err := os.MkdirTemp("", "means-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	c, err := collection.Init(dir, []byte(`{"type":"object"}`))
	if err != nil {
		panic(err)
	}
	doc, err := c.Create(context.Background(), collection.CreateOptions{
		Title: "First note",
		Data:  map[string]any{"status": "draft"},
		Body:  "Written directly from Go.",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(doc.Frontmatter["title"])
	fmt.Println(doc.Body)
	// Output:
	// First note
	// Written directly from Go.
}

func ExampleCollection_Update() {
	dir, err := os.MkdirTemp("", "means-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	c, err := collection.Init(dir, nil)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	doc, err := c.Create(ctx, collection.CreateOptions{Title: "First note"})
	if err != nil {
		panic(err)
	}
	current, err := c.Read(ctx, doc.ID)
	if err != nil {
		panic(err)
	}
	body := "Updated from Go."
	updated, err := c.Update(ctx, current.ID, current.Hash, collection.Patch{Body: &body})
	if err != nil {
		panic(err)
	}
	fmt.Println(updated.Body)
	// Output: Updated from Go.
}

func ExampleCollection_Search() {
	dir, err := os.MkdirTemp("", "means-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	c, err := collection.Init(dir, []byte(`{"type":"object"}`))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	for _, note := range []collection.CreateOptions{
		{Title: "Release checklist", Data: map[string]any{"status": "draft", "created_at": "2026-11-02T08:00:00Z"}, Body: "Check the release.\n"},
		{Title: "Release notes", Data: map[string]any{"status": "published", "created_at": "2026-11-02T08:00:00Z"}},
	} {
		if _, err := c.Create(ctx, note); err != nil {
			panic(err)
		}
	}
	result, err := c.Search(ctx, collection.SearchOptions{
		Query:    "release",
		Filter:   map[string]any{"status": "draft"},
		DateTime: []collection.DateTimeFilter{{Field: "created_at", Op: "on_or_after", Value: "2026-11-01T00:00:00Z"}},
		WithBody: true,
		Limit:    10,
	})
	if err != nil {
		panic(err)
	}
	// A successful search can still report malformed files or duplicate IDs.
	if len(result.Diagnostics) != 0 {
		panic(result.Diagnostics)
	}
	for _, hit := range result.Documents {
		fmt.Println(hit.Frontmatter["title"])
		fmt.Print(*hit.Body) // Non-nil because WithBody was set.
	}
	// Output:
	// Release checklist
	// Check the release.
}

func ExampleSetSchema() {
	dir, err := os.MkdirTemp("", "means-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	c, err := collection.Init(dir, []byte(`true`))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	if _, err := c.Create(ctx, collection.CreateOptions{Title: "Plan", Data: map[string]any{"status": "draft"}}); err != nil {
		panic(err)
	}
	result, err := collection.SetSchema(ctx, c.Path(), []byte(`{"required":["owner"]}`), false)
	if !errors.Is(err, collection.ErrSchemaIncompatible) {
		panic(fmt.Sprintf("expected schema rejection, got %v", err))
	}
	// Rejection returns diagnostics as well as an error, without changing MEANS.md.
	fmt.Printf("Schema rejected: %d document diagnostic\n", len(result.Diagnostics))

	compatible := []byte(`{"type":"object","properties":{"status":{"enum":["draft","ready"]}},"required":["status"]}`)
	result, err = collection.SetSchema(ctx, c.Path(), compatible, false)
	if err != nil {
		panic(err)
	}
	fmt.Println("Installed:", result.Installed)
	// Installation does not change c's snapshot. Reload before the next operation.
	c, err = collection.Load(c.Path())
	if err != nil {
		panic(err)
	}
	if err := c.ValidateData(map[string]any{"status": "draft"}); err != nil {
		panic(err)
	}
	// Output:
	// Schema rejected: 1 document diagnostic
	// Installed: true
}

func ExampleCollection_Update_conflict() {
	dir, err := os.MkdirTemp("", "means-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	c, err := collection.Init(dir, nil)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	created, err := c.Create(ctx, collection.CreateOptions{Title: "Original title"})
	if err != nil {
		panic(err)
	}
	original, err := c.Read(ctx, created.ID)
	if err != nil {
		panic(err)
	}
	// Another caller changes the title after our read.
	if _, err := c.Update(ctx, original.ID, original.Hash, collection.Patch{Frontmatter: map[string]any{"title": "Edited elsewhere"}}); err != nil {
		panic(err)
	}
	body := "My body edit."
	_, err = c.Update(ctx, original.ID, original.Hash, collection.Patch{Body: &body})
	if !errors.Is(err, collection.ErrConflict) {
		panic(fmt.Sprintf("expected conflict, got %v", err))
	}
	current, err := c.Read(ctx, original.ID)
	if err != nil {
		panic(err)
	}
	// Reconcile with the current document. Reapply our edit only if the body is
	// unchanged; otherwise the caller must resolve both body edits first.
	if current.Body != original.Body {
		panic("body changed; resolve the conflicting edits before saving")
	}
	// The body-only patch preserves the other caller's title.
	updated, err := c.Update(ctx, current.ID, current.Hash, collection.Patch{Body: &body})
	if err != nil {
		panic(err) // A new concurrent edit can still cause another conflict.
	}
	fmt.Println(updated.Frontmatter["title"])
	fmt.Println(updated.Body)
	// Output:
	// Edited elsewhere
	// My body edit.
}

func ExampleCollection_SetInstructions() {
	dir, err := os.MkdirTemp("", "means-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	c, err := collection.Init(dir, nil)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	info := c.Describe() // Configuration hash, not a document hash.
	updated, err := c.SetInstructions(ctx, info.Hash, "Review drafts before publishing.\n")
	if err != nil {
		panic(err)
	}
	fmt.Print(updated.Instructions)
	// The receiver still holds the original snapshot; reload after the change.
	c, err = collection.Load(c.Path())
	if err != nil {
		panic(err)
	}
	cleared, err := c.SetInstructions(ctx, c.Describe().Hash, "")
	if err != nil {
		panic(err)
	}
	fmt.Println("Cleared:", cleared.Instructions == "")
	// Output:
	// Review drafts before publishing.
	// Cleared: true
}

func ExampleEmbeddingConfig_WithDefaults() {
	settings, err := (collection.EmbeddingConfig{Provider: "openrouter"}).WithDefaults()
	if err != nil {
		panic(err)
	}
	// This validates settings without network access. WithEmbeddings attaches
	// them to a collection; set OPENROUTER_API_KEY before requests need vectors.
	fmt.Println(settings.Model, settings.Dimensions)
	// Output: openai/text-embedding-3-small 512
}
