package consumer_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/hmans/means/collection"
)

func TestDependencyWorkflow(t *testing.T) {
	ctx := context.Background()
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(ctx, collection.CreateOptions{Title: "From another module", Body: "Hello"})
	if err != nil {
		t.Fatal(err)
	}
	c, err = collection.Load(c.Path())
	if err != nil {
		t.Fatal(err)
	}
	current, err := c.Read(ctx, doc.ID)
	if err != nil || current.Hash != doc.Hash {
		t.Fatalf("read = %+v, %v", current, err)
	}
	listed, err := c.List(ctx, collection.ListOptions{})
	if err != nil || len(listed.Documents) != 1 {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	updated, err := c.Update(ctx, doc.ID, current.Hash, collection.Patch{Frontmatter: map[string]any{"title": "Changed"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Delete(ctx, doc.ID, doc.Hash); !errors.Is(err, collection.ErrConflict) {
		t.Fatalf("stale delete = %v", err)
	}
	if _, err := c.Delete(ctx, doc.ID, updated.Hash); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Read(ctx, doc.ID); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing document = %v", err)
	}
}
