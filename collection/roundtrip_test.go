package collection

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Every value that create and update accept must read back identically.
func TestCreateUpdateReadEquality(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	values := []any{
		nil, true, false, map[string]any{}, []any{},
		json.Number("0"), json.Number("-0"), json.Number("-0.0"), json.Number("1.0"), json.Number("1.50"),
		json.Number("1e5"), json.Number("1E5"), json.Number("1e+5"), json.Number("1E-5"), json.Number("0e0"),
		json.Number("9007199254740993"), json.Number("18446744073709551616"), json.Number("-9223372036854775809"),
		json.Number("123456789012345678901234567890"), json.Number("0.1234567890123456789012345678901234567890"),
		json.Number("1.7976931348623157e308"), json.Number("4.9e-324"), json.Number("1e-400"),
		"1e400", "-1e400", ".inf", ".nan", "0x10", "0o7", "1_000", "~", "null", "y", "NO", "on", "<<", "=",
		"!", "!!str", "&a", "*a", "- a", "? a", ": a", "a: b", "a #b", "#a", "@a", "`a", "%a", "|", ">", "'", "\"",
		"[a]", "{a}", "---", "...", "2026-10-20", "2026-10-20T10:00:00Z", "12:30:45", "0.", ".5", "+1", "1e",
		" a", "a ", "\ta", "a\t", "\n", "a\n", "a\n\n", "\na", "a\r\nb", "a\rb", "\r", "a\n b", " a\nb", "\t\n\t",
		"a\n\tb: c\n\n", "\\", "\\n", "\x00", "\x1b", "\x7f", "\u0080", "\u009f", "\u00a0", "\xef\xbb\xbfa",
		"\ufffe", "\uffff", "🌱", "\U0010ffff",
		"\u0085", "a\u0085b", "\u0085\n\u0085", "a\u2028b", "\u2029\n", "a\n\u2028b: c\n\n",
	}
	for _, value := range values {
		encoded, _ := json.Marshal(value)
		t.Run(string(encoded), func(t *testing.T) {
			title, key := "Edge", "key"
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				title, key = s, s
			}
			data := map[string]any{key: value, "list": []any{value}}
			created, err := c.Create(ctx, CreateOptions{Title: title, Data: data, Body: "Body\n"})
			if err != nil {
				t.Fatal(err)
			}
			assertReadsBack(t, c, created)
			if got := created.Frontmatter["data"]; !reflect.DeepEqual(got, data) || created.Frontmatter["title"] != title {
				t.Errorf("stored %#v, want title %q and data %#v", created.Frontmatter, title, data)
			}
			updated, err := c.Update(ctx, created.ID, created.Hash, Patch{Frontmatter: map[string]any{"data": map[string]any{"again": value}}})
			if err != nil {
				t.Fatal(err)
			}
			assertReadsBack(t, c, updated)
		})
	}
}

func TestCreateUpdateRejectUnrepresentableNumbers(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	doc, err := c.Create(ctx, CreateOptions{Title: "Numbers"})
	if err != nil {
		t.Fatal(err)
	}
	for _, number := range []string{"1e400", "-1e400", "1E400", "1e309", "2e308", "1" + strings.Repeat("0", 400)} {
		data := map[string]any{"n": json.Number(number)}
		const want = "frontmatter.data.n cannot be stored as YAML"
		if _, err := c.Create(ctx, CreateOptions{Title: "Rejected", Data: data}); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Create(%s) = %v, want %q", number, err, want)
		}
		if _, err := c.Update(ctx, doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"data": data}}); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Update(%s) = %v, want %q", number, err, want)
		}
	}
	assertReadsBack(t, c, doc)
	if list, err := c.List(ctx, ListOptions{}); err != nil || len(list.Documents) != 1 {
		t.Errorf("rejected writes left documents behind: %+v, %v", list, err)
	}
}

func assertReadsBack(t *testing.T, c *Collection, doc *Document) {
	t.Helper()
	read, err := c.Read(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("Read() = %v\n%s", err, doc.Markdown)
	}
	if !reflect.DeepEqual(read.Frontmatter, doc.Frontmatter) || read.Body != doc.Body || read.Hash != doc.Hash || read.Markdown != doc.Markdown {
		t.Errorf("read differs from the write response\nresponse: %#v\nread: %#v\n%s", doc.Frontmatter, read.Frontmatter, doc.Markdown)
	}
}

// Encoding either preserves arbitrary strings and numbers or fails.
func FuzzEncodeMarkdownRoundTrip(f *testing.F) {
	for _, seed := range []string{"", "plain", "\u0085", "1e400", "<<", "\t\n\t", "a: b\n", "\x7f", "- a"} {
		f.Add(seed, seed)
	}
	f.Fuzz(func(t *testing.T, text, number string) {
		frontmatter := map[string]any{"title": text, "data": map[string]any{text: []any{text}}}
		if json.Valid([]byte(number)) && number != "" && strings.Trim(number, "-+.eE0123456789") == "" {
			frontmatter["number"] = json.Number(number)
		}
		want, err := normalizeObject(frontmatter)
		if err != nil {
			t.Skip()
		}
		markdown, err := encodeMarkdown(want, "")
		if err != nil {
			return
		}
		got, _, err := parseMarkdown([]byte(markdown))
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("stored %#v (%v), want %#v\n%s", got, err, want, markdown)
		}
	})
}
