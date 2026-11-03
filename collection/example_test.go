package collection_test

import (
	"context"
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
