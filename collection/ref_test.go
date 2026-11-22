package collection

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const refSchema = `{
	"$defs": {"docId": {"type": "string", "x-means-ref": true}},
	"type": "object",
	"properties": {
		"parent": {"$ref": "#/$defs/docId"},
		"blocked_by": {"type": "array", "items": {"$ref": "#/$defs/docId"}},
		"a/b": {"x-means-ref": true},
		"plain": {"x-means-ref": false}
	}
}`

func TestRefKeywordCompilation(t *testing.T) {
	for _, schema := range []string{`{"x-means-ref": "yes"}`, `{"properties": {"p": {"x-means-ref": 1}}}`} {
		if _, err := Init(t.TempDir(), []byte(schema)); err == nil || !strings.Contains(err.Error(), "invalid JSON Schema") {
			t.Fatalf("%s: expected invalid schema, got %v", schema, err)
		}
	}
}

func TestRefKeywordValidatesWrites(t *testing.T) {
	ctx := context.Background()
	c, err := Init(t.TempDir(), []byte(refSchema))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		data map[string]any
		want string
	}{
		{map[string]any{"parent": 42}, "want string"},
		{map[string]any{"parent": "not an id"}, "must be a document ID string"},
		{map[string]any{"blocked_by": []any{"ok", "bad id"}}, "must be a document ID string"},
		{map[string]any{"a/b": 42}, "must be a document ID string"},
	} {
		if _, err := c.Create(ctx, CreateOptions{Title: "Bad", Data: test.data}); !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("%v: expected %q, got %v", test.data, test.want, err)
		}
	}
	if _, err := c.Create(ctx, CreateOptions{Title: "Plain", Data: map[string]any{"plain": 42}}); err != nil {
		t.Fatalf("x-means-ref false must not validate: %v", err)
	}
}

func TestDoctorReportsMissingReferences(t *testing.T) {
	ctx := context.Background()
	c, err := Init(t.TempDir(), []byte(refSchema))
	if err != nil {
		t.Fatal(err)
	}
	parent, err := c.Create(ctx, CreateOptions{Title: "Parent"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := c.Create(ctx, CreateOptions{Title: "Child", Data: map[string]any{
		"parent":     parent.ID,
		"blocked_by": []any{parent.ID, "gone1"},
		"a/b":        "gone2",
		"plain":      "gone3",
	}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := c.Doctor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var messages []string
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.ID != child.ID {
			t.Fatalf("unexpected diagnostic: %+v", diagnostic)
		}
		messages = append(messages, diagnostic.Error)
	}
	want := []string{
		`reference at '/blocked_by/1' points to missing document "gone1"`,
		`reference at '/a~1b' points to missing document "gone2"`,
	}
	if report.Healthy || len(messages) != len(want) {
		t.Fatalf("unexpected report: %+v", report)
	}
	for _, message := range want {
		if !strings.Contains(strings.Join(messages, "\n"), message) {
			t.Fatalf("missing %q in %q", message, messages)
		}
	}

	plain, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plain.Create(ctx, CreateOptions{Title: "Free", Data: map[string]any{"parent": "gone"}}); err != nil {
		t.Fatal(err)
	}
	if report, err := plain.Doctor(ctx); err != nil || !report.Healthy {
		t.Fatalf("schema without references must stay healthy: %+v %v", report, err)
	}
}
