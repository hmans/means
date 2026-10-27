package collection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceMarkdown(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{"type":"object","required":["required"]}`))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(map[string]any{"title": "Before", "data": map[string]any{"required": true, "removed": true}}, "body\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		"---\nid: another\ntitle: After\ndata: {required: true}\n---\nbody\n",
		"---\nid: " + doc.ID + "\ntitle: After\ndata: {}\n---\nbody\n",
		"---\nid: " + doc.ID + "\ntitle: After\ndata: {required: true}\nextra: invalid\n---\nbody\n",
	} {
		if _, err := c.ReplaceMarkdown(doc.ID, doc.Hash, []byte(source)); err == nil {
			t.Fatal("accepted invalid replacement")
		}
		current, err := c.Read(doc.ID)
		if err != nil || current.Hash != doc.Hash {
			t.Fatal("invalid edit changed source", err)
		}
	}
	unchanged, err := c.ReplaceMarkdown(doc.ID, doc.Hash, []byte(doc.Markdown))
	if err != nil || unchanged.Hash != doc.Hash {
		t.Fatal("no-op changed document", err)
	}
	source := "---\nid: " + doc.ID + "\ntitle: After\ndata: {required: null, large: 9007199254740993}\n---\nnew body\n"
	updated, err := c.ReplaceMarkdown(doc.ID, doc.Hash, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if updated.filename != doc.filename || updated.Body != "new body\n" || strings.Contains(updated.Markdown, "removed:") || !strings.Contains(updated.Markdown, "9007199254740993") {
		t.Fatal(updated.Markdown)
	}
	data := updated.Frontmatter["data"].(map[string]any)
	if value, ok := data["required"]; !ok || value != nil {
		t.Fatal(data)
	}
	if _, err := c.ReplaceMarkdown(doc.ID, doc.Hash, []byte(source)); err == nil {
		t.Fatal("accepted stale hash")
	}
	stat, err := os.Stat(filepath.Join(c.Path, updated.filename))
	if err != nil || !stat.Mode().IsRegular() {
		t.Fatal(err)
	}
}
