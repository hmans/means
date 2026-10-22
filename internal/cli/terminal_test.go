package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/means/internal/collection"
)

func TestDocumentTables(t *testing.T) {
	c, err := collection.Init(t.TempDir(), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"First title", "Second title"} {
		if _, err := c.Create(map[string]any{"title": title, "data": map[string]any{"secret": "metadata-marker"}}, "body-marker"); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(c.Path, "00000-broken.md"), []byte("bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"list", "search"} {
		t.Run(command, func(t *testing.T) {
			var out, errs bytes.Buffer
			if err := Execute([]string{command, "--path", c.Path, "--limit", "0"}, nil, &out, &errs, "test"); err != nil {
				t.Fatal(err)
			}
			text := out.String()
			for _, want := range []string{"ID", "TITLE", "First title", "Second title", "Diagnostics", "broken.md"} {
				if !strings.Contains(text, want) {
					t.Errorf("missing %q in %s", want, text)
				}
			}
			for _, unwanted := range []string{"metadata-marker", "body-marker", "\x1b", `"title":`} {
				if strings.Contains(text, unwanted) {
					t.Errorf("unexpected %q in %s", unwanted, text)
				}
			}
			out.Reset()
			if err := Execute([]string{command, "--path", c.Path, "--limit", "1"}, nil, &out, &errs, "test"); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "--limit 0") {
				t.Fatal("missing limit notice")
			}
			if strings.Contains(out.String(), "No documents.") || !strings.Contains(out.String(), "TITLE") || !strings.Contains(out.String(), "00000-broken.md") {
				t.Fatalf("limited output hides valid document or diagnostic: %s", &out)
			}
		})
	}
}

func TestShowDocument(t *testing.T) {
	c, err := collection.Init(t.TempDir(), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(map[string]any{"title": "A *literal* title", "data": map[string]any{"number": json.Number("9007199254740993")}}, "## Heading\n\nA **bold** statement.\n\n- item\n\n```go\nfmt.Println(42)\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if err := Execute([]string{"show", doc.ID, "--path", c.Path}, nil, &out, &errs, "test"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{doc.ID, "A *literal* title", "Heading", "bold", "item", "fmt.Println(42)"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q: %s", want, &out)
		}
	}
	for _, unwanted := range []string{"\x1b", "**bold**", "```go", "frontmatter", "9007199254740993"} {
		if strings.Contains(out.String(), unwanted) {
			t.Errorf("unexpected %q: %s", unwanted, &out)
		}
	}
	var read bytes.Buffer
	out.Reset()
	if err := Execute([]string{"show", doc.ID, "--path", c.Path, "--json"}, nil, &out, &errs, "test"); err != nil {
		t.Fatal(err)
	}
	if err := Execute([]string{"read", doc.ID, "--path", c.Path, "--json"}, nil, &read, &errs, "test"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), read.Bytes()) {
		t.Fatal("show JSON differs from read JSON")
	}
	for _, args := range [][]string{{"show"}, {"show", "missing"}, {"show", doc.ID, "extra"}} {
		out.Reset()
		errs.Reset()
		if err := Execute(append(args, "--path", c.Path, "--json"), nil, &out, &errs, "test"); err == nil || out.Len() != 0 || !json.Valid(errs.Bytes()) {
			t.Fatalf("invalid show: %v %s %s", err, &out, &errs)
		}
	}
	out.Reset()
	errs.Reset()
	if err := Execute([]string{"show", "--help", "--json"}, nil, &out, &errs, "test"); err != nil || !json.Valid(out.Bytes()) || !strings.Contains(out.String(), "read --json") {
		t.Fatalf("help: %v %s", err, &out)
	}
}

func TestTableWrappingAndEmptyOutput(t *testing.T) {
	var out bytes.Buffer
	text := documentTable(&out, [][]string{{"abc12", terminalCell(strings.Repeat("長い title ", 30) + "\n\x1b[31mred\x1b[0m")}}, nil, false, "documents")
	for _, line := range strings.Split(text, "\n") {
		if ansi.StringWidth(line) > 80 {
			t.Fatalf("line too wide: %q", line)
		}
	}
	if !strings.Contains(ansi.Strip(text), "red") {
		t.Fatal("title truncated")
	}
	if got := documentTable(&out, nil, nil, false, "matches"); got != "No matches.\n" {
		t.Fatal(got)
	}
	t.Setenv("NO_COLOR", "1")
	if err := writeTerminal(&out, text); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Fatal("escape codes in plain output")
	}
}

func TestShowProseWrapAndBulletColor(t *testing.T) {
	doc := &collection.Document{
		ID: "wrap1", Frontmatter: map[string]any{"title": "A long document"},
		Body: strings.Repeat("Words in a paragraph. ", 20) + "\n\n- " + strings.Repeat("Words in a bullet. ", 20) + "\n  - Nested item\n",
	}
	for _, width := range []int{24, 40, 80, 160} {
		text, err := renderDocumentWidth(width, doc)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(text, "\n") {
			if ansi.StringWidth(line) > min(width, 80) {
				t.Fatalf("width %d overflow: %q", width, line)
			}
		}
		if !strings.Contains(text, "\x1b[38;5;37m• ") {
			t.Fatalf("bullet has no accent: %q", text)
		}
		plain := ansi.Strip(text)
		if strings.Count(plain, "Words") != 40 || !strings.Contains(plain, "Nested item") {
			t.Fatal("wrapping lost content")
		}
	}
}
