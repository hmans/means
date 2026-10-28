package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTUIWatcher(t *testing.T) {
	root := t.TempDir()
	w, err := newTUIWatcher(root)
	if err != nil {
		t.Fatal(err)
	}
	defer w.close()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	changed := func() {
		t.Helper()
		select {
		case msg := <-w.messages:
			if _, ok := msg.(tuiChanged); !ok {
				t.Fatalf("unexpected watch message: %#v", msg)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("filesystem change was not reported")
		}
	}
	quiet := func() {
		t.Helper()
		select {
		case msg := <-w.messages:
			t.Fatalf("unexpected notification: %#v", msg)
		case <-time.After(2 * tuiWatchDelay):
		}
	}
	// A burst of writes produces one notification after the last write.
	for i := 0; i < 5; i++ {
		write("document.md", "body")
		time.Sleep(20 * time.Millisecond)
	}
	changed()
	quiet()
	// Atomic replacement is tracked by the parent directory watch.
	write(".temporary", "replacement")
	if err := os.Rename(filepath.Join(root, ".temporary"), filepath.Join(root, "document.md")); err != nil {
		t.Fatal(err)
	}
	changed()
	if err := os.Remove(filepath.Join(root, "document.md")); err != nil {
		t.Fatal(err)
	}
	changed()
	if err := os.Mkdir(filepath.Join(root, "child"), 0700); err != nil {
		t.Fatal(err)
	}
	write("child/new.md", "new")
	changed()
	write("child/new.md", "updated")
	changed()
	write("child/MEANS.md", "boundary")
	changed()
	write("child/new.md", "ignored nested document")
	write(".hidden.md", "ignored hidden document")
	quiet()
	if err := os.Remove(filepath.Join(root, "child/MEANS.md")); err != nil {
		t.Fatal(err)
	}
	changed()
	write("child/new.md", "owned again")
	changed()
}

func TestTUIRefreshQueuesChangesDuringLoad(t *testing.T) {
	c := tuiTestCollection(t)
	m := newTUIModel(c.Path)
	initial := m.Init()
	m.Update(tuiChanged{})
	_, followup := m.Update(initial())
	if followup == nil || !m.refreshing {
		t.Fatal("change during load was lost")
	}
	m.Update(followup())
	if m.refreshing || m.refreshPending {
		t.Fatal("queued refresh did not finish")
	}
}
