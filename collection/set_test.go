package collection

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSetEditsFields(t *testing.T) {
	ctx := context.Background()
	c, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(ctx, CreateOptions{Title: "Note", Data: map[string]any{"keep": nil, "tags": []any{"a"}}, Body: "Body\n"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := c.Set(ctx, doc.ID, "",
		FieldEdit{Path: []string{"title"}, Value: "Renamed"},
		FieldEdit{Path: []string{"data", "status"}, Value: "done"},
		FieldEdit{Path: []string{"data", "owner", "name"}, Value: "Ada"},
		FieldEdit{Path: []string{"data", "tags"}, Op: FieldAdd, Value: "b"},
		FieldEdit{Path: []string{"data", "tags"}, Op: FieldAdd, Value: "a"},
		FieldEdit{Path: []string{"data", "tags"}, Op: FieldRemove, Value: "a"},
		FieldEdit{Path: []string{"data", "list"}, Op: FieldAdd, Value: 1},
		FieldEdit{Path: []string{"data", "gone"}, Op: FieldRemove, Value: 1},
		FieldEdit{Path: []string{"data", "missing", "child"}, Value: nil},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"keep":   nil,
		"status": "done",
		"owner":  map[string]any{"name": "Ada"},
		"tags":   []any{"b"},
		"list":   []any{json.Number("1")},
	}
	stored, err := c.Read(ctx, doc.ID)
	if err != nil || stored.Hash != updated.Hash || stored.Frontmatter["title"] != "Renamed" || stored.Body != "Body\n" || !reflect.DeepEqual(stored.Frontmatter["data"], want) {
		t.Fatalf("incorrect set: %#v %v", stored, err)
	}
	if _, err := c.Set(ctx, doc.ID, "", FieldEdit{Path: []string{"data", "owner", "name"}, Value: nil}); err != nil {
		t.Fatal(err)
	}
	stored, _ = c.Read(ctx, doc.ID)
	if !reflect.DeepEqual(stored.Frontmatter["data"].(map[string]any)["owner"], map[string]any{}) {
		t.Fatalf("nested delete failed: %#v", stored.Frontmatter)
	}
	if _, err := c.Set(ctx, doc.ID, doc.Hash, FieldEdit{Path: []string{"data", "x"}, Value: 1}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if _, err := c.Set(ctx, doc.ID, stored.Hash, FieldEdit{Path: []string{"data", "x"}, Value: 1}); err != nil {
		t.Fatalf("current hash rejected: %v", err)
	}
}

func TestSetRejectsInvalidEdits(t *testing.T) {
	ctx := context.Background()
	c, err := Init(t.TempDir(), []byte(`{"type":"object","properties":{"n":{"type":"integer"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(ctx, CreateOptions{Title: "Note", Data: map[string]any{"s": "text"}})
	if err != nil {
		t.Fatal(err)
	}
	for name, test := range map[string]struct {
		edits []FieldEdit
		want  string
	}{
		"none":         {nil, "at least one"},
		"id":           {[]FieldEdit{{Path: []string{"id"}, Value: "other"}}, "title or data"},
		"empty path":   {[]FieldEdit{{}}, "title or data"},
		"nested title": {[]FieldEdit{{Path: []string{"title", "x"}, Value: "y"}}, "no nested"},
		"empty name":   {[]FieldEdit{{Path: []string{"data", ""}, Value: 1}}, "empty name"},
		"non-object":   {[]FieldEdit{{Path: []string{"data", "s", "x"}, Value: 1}}, "not an object"},
		"non-array":    {[]FieldEdit{{Path: []string{"data", "s"}, Op: FieldAdd, Value: 1}}, "not an array"},
		"add null":     {[]FieldEdit{{Path: []string{"data", "s"}, Op: FieldAdd}}, "null"},
		"schema":       {[]FieldEdit{{Path: []string{"data", "n"}, Value: "x"}}, "schema"},
		"blank title":  {[]FieldEdit{{Path: []string{"title"}, Value: " "}}, "title"},
		"delete data":  {[]FieldEdit{{Path: []string{"data"}}}, "data"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := c.Set(ctx, doc.ID, "", test.edits...); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
			stored, err := c.Read(ctx, doc.ID)
			if err != nil || stored.Hash != doc.Hash {
				t.Fatalf("failed edit changed document: %v", err)
			}
		})
	}
}
