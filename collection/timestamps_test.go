package collection

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fakeClock(t *testing.T) func(time.Duration) {
	t.Helper()
	current := time.Date(2026, 11, 22, 10, 0, 0, 0, time.FixedZone("CET", 3600))
	previous := clock
	clock = func() time.Time { return current }
	t.Cleanup(func() { clock = previous })
	return func(d time.Duration) { current = current.Add(d) }
}

func TestTimestampsMaintainedOnWrites(t *testing.T) {
	ctx := context.Background()
	advance := fakeClock(t)
	c, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(ctx, CreateOptions{Title: "Note", Body: "Body\n"})
	if err != nil {
		t.Fatal(err)
	}
	created := "2026-11-22T09:00:00Z"
	check := func(doc *Document, updated string) {
		t.Helper()
		if doc.Frontmatter[CreatedAtField] != created || doc.Frontmatter[UpdatedAtField] != updated {
			t.Fatalf("timestamps = %v, %v; want %s, %s", doc.Frontmatter[CreatedAtField], doc.Frontmatter[UpdatedAtField], created, updated)
		}
		stored, err := c.Read(ctx, doc.ID)
		if err != nil || !reflect.DeepEqual(stored.Frontmatter, doc.Frontmatter) {
			t.Fatalf("stored %v differs from returned %v: %v", stored.Frontmatter, doc.Frontmatter, err)
		}
	}
	check(doc, created)

	advance(time.Minute)
	noop, err := c.Update(ctx, doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"title": "Note"}})
	if err != nil || noop.Hash != doc.Hash {
		t.Fatalf("no-op update rewrote document: %v", err)
	}
	doc, err = c.Update(ctx, doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"data": map[string]any{"a": 1}}})
	if err != nil {
		t.Fatal(err)
	}
	check(doc, "2026-11-22T09:01:00Z")

	advance(time.Minute)
	if doc, err = c.Append(ctx, doc.ID, "", "More\n"); err != nil {
		t.Fatal(err)
	}
	check(doc, "2026-11-22T09:02:00Z")

	advance(time.Minute)
	if doc, err = c.Set(ctx, doc.ID, "", FieldEdit{Path: []string{"data", "a"}, Value: 2}); err != nil {
		t.Fatal(err)
	}
	check(doc, "2026-11-22T09:03:00Z")

	// Edited timestamp values are ignored; means keeps created_at and bumps updated_at.
	advance(time.Minute)
	edited := strings.Replace(doc.Markdown, created, "2000-01-01T00:00:00Z", 1)
	edited = strings.Replace(edited, "title: Note", "title: Edited", 1)
	if doc, err = c.ReplaceMarkdown(ctx, doc.ID, doc.Hash, []byte(edited)); err != nil {
		t.Fatal(err)
	}
	check(doc, "2026-11-22T09:04:00Z")
	if doc.Frontmatter["title"] != "Edited" {
		t.Fatalf("edit not applied: %v", doc.Frontmatter)
	}
}

func TestTimestampsAreNotClientWritable(t *testing.T) {
	ctx := context.Background()
	c, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.create(ctx, map[string]any{"title": "x", CreatedAtField: "2026-01-01T00:00:00Z"}, ""); err == nil || !strings.Contains(err.Error(), "maintained by means") {
		t.Fatalf("create accepted created_at: %v", err)
	}
	doc, err := c.Create(ctx, CreateOptions{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{CreatedAtField, UpdatedAtField} {
		if _, err := c.Update(ctx, doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{key: nil}}); err == nil || !strings.Contains(err.Error(), "maintained by means") {
			t.Fatalf("update accepted %s: %v", key, err)
		}
		if _, err := c.Set(ctx, doc.ID, "", FieldEdit{Path: []string{key}, Value: "2026-01-01T00:00:00Z"}); err == nil {
			t.Fatalf("set accepted %s", key)
		}
	}
}

func TestTimestampsOnLegacyAndInvalidDocuments(t *testing.T) {
	ctx := context.Background()
	fakeClock(t)
	c, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, source string) {
		if err := os.WriteFile(filepath.Join(c.Path(), name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("legacy.md", "---\nid: legacy\ntitle: Legacy\ndata: {}\n---\n")
	write("offset.md", "---\nid: offset\ntitle: Offset\ndata: {}\ncreated_at: 2026-11-22T11:30:00+02:00\n---\n")
	write("bad.md", "---\nid: bad\ntitle: Bad\ndata: {}\nupdated_at: yesterday\n---\n")
	write("number.md", "---\nid: number\ntitle: Number\ndata: {}\ncreated_at: 5\n---\n")

	legacy, err := c.Read(ctx, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := c.Set(ctx, "legacy", "", FieldEdit{Path: []string{"data", "a"}, Value: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, invented := updated.Frontmatter[CreatedAtField]; invented || updated.Frontmatter[UpdatedAtField] != "2026-11-22T09:00:00Z" || legacy.Frontmatter[UpdatedAtField] != nil {
		t.Fatalf("legacy timestamps: %v", updated.Frontmatter)
	}
	report, err := c.Doctor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bad := map[string]bool{}
	for _, diagnostic := range report.Diagnostics {
		bad[diagnostic.ID] = true
	}
	if !bad["bad"] || !bad["number"] || bad["offset"] || bad["legacy"] {
		t.Fatalf("diagnostics: %+v", report.Diagnostics)
	}

	// Sorting compares instants; Expr sees the top-level fields.
	ids, _ := listIDs(t, c, ListOptions{Sort: []string{"-" + UpdatedAtField}})
	if !reflect.DeepEqual(ids, []string{"legacy", "offset"}) {
		t.Fatalf("sort by updated_at: %v", ids)
	}
	ids, _ = listIDs(t, c, ListOptions{Sort: []string{CreatedAtField}, Where: `created_at != nil and date(created_at) < date("2026-11-22T10:00:00Z")`})
	if !reflect.DeepEqual(ids, []string{"offset"}) {
		t.Fatalf("where on created_at: %v", ids)
	}
}
