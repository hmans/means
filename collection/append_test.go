package collection

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

func TestAppendPreservesSource(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(c.Path(), "manual.md")
	for _, original := range []string{
		"---\r\n# keep comment\r\nid: manual\r\ndata: {}\r\ntitle: 'Manual'\r\n---\r\nOld body",
		"---\nid: manual\ndata: {}\ntitle: Manual\n---\n",
		"---\nid: manual\ndata: {}\ntitle: Manual\n---",
	} {
		if err := os.WriteFile(filename, []byte(original), 0o640); err != nil {
			t.Fatal(err)
		}
		current, err := c.Read(context.Background(), "manual")
		if err != nil {
			t.Fatal(err)
		}
		noop, err := c.Append(context.Background(), current.ID, current.Hash, "")
		if err != nil || noop.Hash != current.Hash || noop.Markdown != original {
			t.Fatalf("empty append changed file: %+v %v", noop, err)
		}
		for _, text := range []string{" inline", "\r\n\r\n## More\r\nText with ü.\r\n"} {
			updated, err := c.Append(context.Background(), current.ID, current.Hash, text)
			if err != nil {
				t.Fatal(err)
			}
			prefix := current.Markdown[:len(current.Markdown)-len(current.Body)]
			if !strings.HasSuffix(prefix, "\n") {
				prefix += "\n"
			}
			want := prefix + current.Body + text
			stored, err := os.ReadFile(filename)
			if err != nil || string(stored) != want || updated.Markdown != want || updated.Body != current.Body+text || updated.Hash != contentHash(want) {
				t.Fatalf("incorrect append: %+v %v; stored %q", updated, err, stored)
			}
			stat, err := os.Stat(filename)
			if err != nil || stat.Mode().Perm() != 0o640 {
				t.Fatalf("permissions changed: %v %v", stat, err)
			}
			if _, err := c.Append(context.Background(), current.ID, current.Hash, "replay"); !errors.Is(err, ErrConflict) {
				t.Fatalf("expected conflict, got %v", err)
			}
			current = updated
		}
	}
	assertNoUpdateArtifacts(t, c.Path())
}

func TestAppendFailuresPreserveSource(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(context.Background(), CreateOptions{Title: "Keep", Body: "Original"})
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tt := range []struct {
		name, id, hash, text string
		ctx                  context.Context
	}{
		{"invalid hash", doc.ID, "bad", "x", context.Background()},
		{"stale hash", doc.ID, strings.Repeat("0", 64), "x", context.Background()},
		{"invalid UTF-8", doc.ID, doc.Hash, "\xff", context.Background()},
		{"missing document", "missing", doc.Hash, "x", context.Background()},
		{"path", "../outside", doc.Hash, "x", context.Background()},
		{"nil context", doc.ID, doc.Hash, "x", nil},
		{"canceled context", doc.ID, doc.Hash, "x", canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := c.Append(tt.ctx, tt.id, tt.hash, tt.text); err == nil {
				t.Fatal("invalid append succeeded")
			}
			stored, err := os.ReadFile(filepath.Join(c.Path(), doc.filename))
			if err != nil || string(stored) != doc.Markdown {
				t.Fatalf("failed append changed source: %q %v", stored, err)
			}
		})
	}
	if _, err := SetSchema(context.Background(), c.Path(), []byte("false"), true); err != nil {
		t.Fatal(err)
	}
	c, err = Load(c.Path())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Append(context.Background(), doc.ID, doc.Hash, "x"); !errors.Is(err, ErrValidation) {
		t.Fatalf("schema-invalid append: %v", err)
	}
	if err := os.WriteFile(filepath.Join(c.Path(), "duplicate.md"), []byte(doc.Markdown), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Append(context.Background(), doc.ID, doc.Hash, "x"); !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("duplicate append: %v", err)
	}
	stored, err := os.ReadFile(filepath.Join(c.Path(), doc.filename))
	if err != nil || string(stored) != doc.Markdown {
		t.Fatalf("failed append changed source: %q %v", stored, err)
	}
	assertNoUpdateArtifacts(t, c.Path())
}

func TestAppendEmbeddingMaintenance(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	doc, err := c.Create(context.Background(), CreateOptions{Title: "Note", Body: "Original"})
	if err != nil {
		t.Fatal(err)
	}
	doc, err = c.Append(context.Background(), doc.ID, "", "\nAddition")
	if err != nil {
		t.Fatal(err)
	}
	if len(api.inputs) != 2 || api.inputs[1] != "Title: Note\n\nOriginal\nAddition" || embeddingRecords(t, c)[doc.ID].Hash != contentHash(embeddingInput(doc.Frontmatter, doc.Body)) {
		t.Fatalf("append did not refresh embedding: %v", api.inputs)
	}
	api.status = 503
	if _, err := c.Append(context.Background(), doc.ID, "", "failed"); err == nil {
		t.Fatal("embedding failure ignored")
	}
	after, err := c.Read(context.Background(), doc.ID)
	if err != nil || after.Hash != doc.Hash {
		t.Fatalf("embedding failure changed source: %+v %v", after, err)
	}
	api.status = 0
	api.hook = func() {
		if err := os.WriteFile(filepath.Join(c.Path(), doc.filename), []byte(doc.Markdown+"external"), 0o600); err != nil {
			t.Error(err)
		}
	}
	if _, err := c.Append(context.Background(), doc.ID, "", "racing"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict after API call, got %v", err)
	}
	after, err = c.Read(context.Background(), doc.ID)
	if err != nil || after.Body != doc.Body+"external" {
		t.Fatalf("lost external edit: %+v %v", after, err)
	}
}

func TestAppendProcesses(t *testing.T) {
	if dir := os.Getenv("MEANS_APPEND_TEST_PATH"); dir != "" {
		c, err := Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		for range 5 {
			if _, err := c.Append(context.Background(), "shared", "", "addition\n"); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Path(), "shared.md"), []byte("---\nid: shared\ntitle: Shared\ndata: {}\n---\nOriginal\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var results []<-chan error
	for range 4 {
		cmd := exec.Command(os.Args[0], "-test.run=^TestAppendProcesses$")
		cmd.Env = append(os.Environ(), "MEANS_APPEND_TEST_PATH="+c.Path())
		var output strings.Builder
		cmd.Stdout, cmd.Stderr = &output, &output
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		result := make(chan error, 1)
		results = append(results, result)
		go func() {
			err := cmd.Wait()
			if err != nil {
				t.Errorf("child append failed: %v\n%s", err, output.String())
			}
			result <- err
		}()
	}
	for _, result := range results {
		<-result
	}
	doc, err := c.Read(context.Background(), "shared")
	if err != nil || doc.Body != "Original\n"+strings.Repeat("addition\n", 20) {
		t.Fatalf("lost concurrent append: %+v %v", doc, err)
	}
	assertNoUpdateArtifacts(t, c.Path())
}

func TestAppendLockCancellation(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(context.Background(), CreateOptions{Title: "Note", Body: "Original"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(c.Path(), ".means"), 0o700); err != nil {
		t.Fatal(err)
	}
	lock := flock.New(filepath.Join(c.Path(), ".means", "append.lock"))
	if err := lock.Lock(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lock.Unlock() })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.Append(ctx, doc.ID, "", "blocked"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected canceled lock wait, got %v", err)
	}
	stored, err := c.Read(context.Background(), doc.ID)
	if err != nil || stored.Hash != doc.Hash {
		t.Fatalf("canceled append changed source: %+v %v", stored, err)
	}
	if err := lock.Unlock(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Append(context.Background(), doc.ID, "", " succeeds"); err != nil {
		t.Fatalf("failed to append after lock released: %v", err)
	}
}

func TestAppendRejectsLockSymlinks(t *testing.T) {
	for _, name := range []string{".means", ".means/append.lock"} {
		t.Run(name, func(t *testing.T) {
			c, err := Init(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := c.Create(context.Background(), CreateOptions{Title: "Note", Body: "Original"})
			if err != nil {
				t.Fatal(err)
			}
			target := t.TempDir()
			if name != ".means" {
				if err := os.MkdirAll(filepath.Join(c.Path(), ".means"), 0o700); err != nil {
					t.Fatal(err)
				}
				target = filepath.Join(target, "target")
				if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(target, filepath.Join(c.Path(), name)); err != nil {
				t.Fatal(err)
			}
			if _, err := c.Append(context.Background(), doc.ID, "", "rejected"); err == nil {
				t.Fatal("symlink accepted")
			}
			stored, err := c.Read(context.Background(), doc.ID)
			if err != nil || stored.Hash != doc.Hash {
				t.Fatalf("rejected append changed source: %+v %v", stored, err)
			}
		})
	}
}
