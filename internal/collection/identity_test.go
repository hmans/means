package collection

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReservedEnvelopeAndDataSchema(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{"type":"object","properties":{"id":{"type":"integer"},"title":{"type":"boolean"}},"required":["id","title"],"additionalProperties":false}`))
	if err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"title": "Means title", "data": map[string]any{"id": 42, "title": true}}
	doc, err := c.Create(input, "Body")
	if err != nil {
		t.Fatal(err)
	}
	if input["id"] != nil || doc.Frontmatter["id"] != doc.ID || doc.Frontmatter["title"] != "Means title" {
		t.Fatalf("invalid identity or mutated input: %+v", doc)
	}
	data := doc.Frontmatter["data"].(map[string]any)
	if data["id"] != json.Number("42") || data["title"] != true {
		t.Fatalf("application field collision: %v", data)
	}
	for _, fm := range []map[string]any{
		{"id": "supplied", "title": "x", "data": data},
		{"id": nil, "title": "x", "data": data},
		{"title": " ", "data": data},
		{"title": "x", "data": nil},
		{"title": "x", "data": []any{}},
		{"title": "x", "data": map[string]any{}},
		{"title": "x", "status": "open", "data": data},
	} {
		if _, err := c.Create(fm, ""); err == nil {
			t.Fatalf("accepted invalid input: %v", fm)
		}
	}
	for _, patch := range []map[string]any{
		{"id": doc.ID}, {"id": "changed"}, {"id": nil}, {"title": nil}, {"title": " "},
		{"data": nil}, {"data": 42}, {"unknown": nil}, {"data": map[string]any{"id": nil}},
	} {
		if _, err := c.Update(doc.ID, doc.Hash, Patch{Frontmatter: patch}); err == nil {
			t.Fatalf("accepted invalid patch: %v", patch)
		}
		current, err := c.Read(doc.ID)
		if err != nil || current.Markdown != doc.Markdown {
			t.Fatalf("failed patch changed original: %v", err)
		}
	}
	changed, err := c.Update(doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"data": map[string]any{"id": 43}}})
	if err != nil || changed.ID != doc.ID {
		t.Fatalf("application ID cannot be patched independently: %+v %v", changed, err)
	}
	assertNoUpdateArtifacts(t, c.Path)
}

func TestIdentitySurvivesRenameMoveAndTitleEdit(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(map[string]any{"title": "Add German translations!"}, "Keep body\n")
	if err != nil {
		t.Fatal(err)
	}
	if doc.filename != doc.ID+"-add-german-translations.md" {
		t.Fatal(doc.filename)
	}
	nested := filepath.Join(c.Path, "translations")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(nested, "a completely different name.md")
	if err := os.Rename(filepath.Join(c.Path, doc.filename), moved); err != nil {
		t.Fatal(err)
	}
	read, err := c.Read(doc.ID)
	if err != nil || read.Hash != doc.Hash || read.Markdown != doc.Markdown {
		t.Fatalf("move broke read: %+v %v", read, err)
	}
	result, err := c.List(0)
	if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != doc.ID {
		t.Fatalf("move broke list: %+v %v", result, err)
	}
	updated, err := c.Update(doc.ID, read.Hash, Patch{Frontmatter: map[string]any{"title": "New title"}})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(moved)
	if err != nil || string(stored) != updated.Markdown || updated.Body != doc.Body || updated.ID != doc.ID {
		t.Fatalf("title edit changed identity/path/body: %+v %v", updated, err)
	}
	if _, err := c.Read("a completely different name"); err == nil {
		t.Fatal("filename accepted as identity")
	}
	if _, err := c.Update(doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"title": "Stale"}}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected stale conflict: %v", err)
	}
	assertNoUpdateArtifacts(t, nested)
}

func TestDuplicateIDsAcrossPaths(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(map[string]any{"title": "Original"}, "Keep")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(c.Path, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Even an invalid envelope with the same parsed ID is ambiguous.
	for _, contents := range []string{doc.Markdown, strings.Replace(doc.Markdown, "title: Original", "title: null", 1)} {
		if err := os.WriteFile(filepath.Join(c.Path, "nested", "copy.md"), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Read(doc.ID); !errors.Is(err, ErrDuplicateID) {
			t.Fatalf("read did not reject duplicate: %v", err)
		}
		if _, err := c.Update(doc.ID, doc.Hash, Patch{Frontmatter: map[string]any{"title": "Changed"}}); !errors.Is(err, ErrDuplicateID) {
			t.Fatalf("update did not reject duplicate: %v", err)
		}
		for _, limit := range []int{1, 0} {
			result, err := c.List(limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Documents) != 0 || len(result.Diagnostics) != 2 || result.HasMore {
				t.Fatalf("unexpected duplicate listing: %+v", result)
			}
			for _, summary := range result.Diagnostics {
				if summary.ID != doc.ID || summary.Path == "" || !strings.Contains(summary.Error, "duplicate document ID") {
					t.Fatalf("ambiguous summary: %+v", summary)
				}
			}
		}
		original, err := os.ReadFile(filepath.Join(c.Path, doc.filename))
		if err != nil || string(original) != doc.Markdown {
			t.Fatal("ambiguous update changed original")
		}
	}
}

func TestCandidateCollisionAfterRename(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	fm := map[string]any{"title": "First", "data": map[string]any{}}
	doc, err := createCandidate(root, "samecandidate", fm, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Rename(doc.filename, "manually-renamed.md"); err != nil {
		t.Fatal(err)
	}
	fm["title"] = "Different title"
	if _, err := createCandidate(root, "samecandidate", fm, "", nil); !errors.Is(err, os.ErrExist) {
		t.Fatalf("ID collision not rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(c.Path, ".means-create-reserved.tmp"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := createCandidate(root, "reserved", fm, "", nil); !errors.Is(err, os.ErrExist) {
		t.Fatalf("allocation collision not rejected: %v", err)
	}
	if err := root.Remove(".means-create-reserved.tmp"); err != nil {
		t.Fatal(err)
	}
	assertNoUpdateArtifacts(t, c.Path)
}

func TestCandidateAllocationAcrossProcesses(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var children []*exec.Cmd
	for range 6 {
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestCandidateProcessHelper$")
		cmd.Env = append(os.Environ(), "MEANS_CREATE_TEST_ROOT="+c.Path)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		children = append(children, cmd)
	}
	successes := 0
	for _, cmd := range children {
		err := cmd.Wait()
		if err == nil {
			successes++
			continue
		}
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 3 {
			t.Fatalf("unexpected child failure: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("same candidate allocated %d times", successes)
	}
	listed, err := c.List(0)
	if err != nil || len(listed.Documents) != 1 || len(listed.Diagnostics) != 0 {
		t.Fatalf("invalid stored allocation: %+v %v", listed, err)
	}
	assertNoUpdateArtifacts(t, c.Path)
}

func TestCandidateProcessHelper(t *testing.T) {
	dir := os.Getenv("MEANS_CREATE_TEST_ROOT")
	if dir == "" {
		return
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		os.Exit(2)
	}
	// Distinct titles ensure filename O_EXCL alone could not prevent duplication.
	_, err = createCandidate(root, "samecandidate", map[string]any{"title": fmt.Sprint(os.Getpid()), "data": map[string]any{}}, "", nil)
	root.Close()
	if errors.Is(err, os.ErrExist) {
		os.Exit(3)
	}
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestTitleSlugs(t *testing.T) {
	for title, want := range map[string]string{
		"Add German translations!": "add-german-translations", " ../../A\\B ": "a-b",
		"こんにちは 世界": "こんにちは-世界", "✨": "document", "---": "document",
		strings.Repeat("A", 200): strings.Repeat("a", 80),
	} {
		if got := titleSlug(title); got != want {
			t.Errorf("slug(%q)=%q, want %q", title, got, want)
		}
	}
}
