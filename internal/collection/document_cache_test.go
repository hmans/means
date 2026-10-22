package collection

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDocumentCacheMatchesSource(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	cache := filepath.Join(t.TempDir(), "documents.jsonl")
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(c.Path, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	check := func() {
		t.Helper()
		want, err := scanDocuments(root)
		if err != nil {
			t.Fatal(err)
		}
		got, err := scanCachedDocuments(root, cache)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatalf("entries: %d != %d", len(got), len(want))
		}
		for i := range want {
			if !reflect.DeepEqual(got[i].doc, want[i].doc) || got[i].filename != want[i].filename || errorText(got[i].err) != errorText(want[i].err) {
				t.Fatalf("cached scan differs at %s: %+v != %+v", want[i].filename, got[i], want[i])
			}
		}
	}
	content := "---\r\nid: exact\r\ntitle: Unicode Über\r\ndata:\r\n  number: 9007199254740993\r\n  decimal: 1.234567890123456789\r\n  nested: [true, null, {key: value}]\r\n---\r\nBody text."
	write("a.md", content)
	write("bad.md", "not frontmatter")
	write("invalid.md", "---\nid: invalid\ntitle: 42\ndata: {}\n---\nbody")
	check() // cold
	before, err := os.Stat(cache)
	if err != nil {
		t.Fatal(err)
	}
	check() // warm, including exact numbers and diagnostics
	after, err := os.Stat(cache)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("unchanged cache was replaced")
	}
	info, err := os.Stat(filepath.Join(c.Path, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	write("a.md", strings.Replace(content, "Body", "Edit", 1))
	if err := os.Chtimes(filepath.Join(c.Path, "a.md"), info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	check()
	if err := os.Rename(filepath.Join(c.Path, "a.md"), filepath.Join(c.Path, "moved.md")); err != nil {
		t.Fatal(err)
	}
	write("bad.md", content) // repaired, duplicate ID
	check()
	if err := os.Remove(filepath.Join(c.Path, "moved.md")); err != nil {
		t.Fatal(err)
	}
	check()
	for _, damage := range []string{"", "{bad json", "{\"version\":999}\n", "{\"version\":1}\n{}\n"} {
		if err := os.WriteFile(cache, []byte(damage), 0o600); err != nil {
			t.Fatal(err)
		}
		check()
	}
	data, err := os.ReadFile(cache)
	if err != nil {
		t.Fatal(err)
	}
	// Valid JSON can still be corrupt; detect edited records and truncation.
	for _, damaged := range [][]byte{bytes.Replace(data, []byte("Unicode"), []byte("Changed"), 1), data[:bytes.LastIndexByte(data[:len(data)-1], '\n')+1]} {
		if err := os.WriteFile(cache, damaged, 0o600); err != nil {
			t.Fatal(err)
		}
		check()
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestDocumentCacheEmptyAndInvalidTransitions(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	cache := filepath.Join(t.TempDir(), "documents.jsonl")
	for _, step := range []struct {
		content string
		records int
	}{
		{"", 0},
		{"invalid", 0},
		{"---\nid: valid\ntitle: Valid\ndata: {}\n---\nbody", 1},
		{"invalid again", 0},
		{"", 0},
	} {
		if step.content != "" {
			if err := os.WriteFile(filepath.Join(c.Path, "doc.md"), []byte(step.content), 0o600); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.Remove(filepath.Join(c.Path, "doc.md")); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
		}
		if _, err := scanCachedDocuments(root, cache); err != nil {
			t.Fatal(err)
		}
		records, err := loadDocumentCache(cache)
		if err != nil || records == nil || len(records) != step.records {
			t.Fatalf("records: %+v, %v", records, err)
		}
		before, err := os.Stat(cache)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := scanCachedDocuments(root, cache); err != nil {
			t.Fatal(err)
		}
		after, err := os.Stat(cache)
		if err != nil {
			t.Fatal(err)
		}
		if !os.SameFile(before, after) {
			t.Fatal("unchanged cache replaced")
		}
	}
}

func TestDocumentCacheRejectsNonregularPaths(t *testing.T) {
	for _, kind := range []string{"symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			c, err := Init(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(c.Path, ".means"), 0o700); err != nil {
				t.Fatal(err)
			}
			cache := filepath.Join(c.Path, ".means", "documents.jsonl")
			if kind == "symlink" {
				err = os.Symlink(filepath.Join(t.TempDir(), "missing"), cache)
			} else {
				err = os.Mkdir(cache, 0o700)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Search(SearchOptions{Query: "text"}); err == nil {
				t.Fatal("accepted nonregular cache")
			}
		})
	}
}
