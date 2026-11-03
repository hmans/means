package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTUIRefreshQueuesChangesDuringLoad(t *testing.T) {
	c := tuiTestCollection(t)
	m := newTUIModel(c.Path())
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

func TestTUIWatcherAdapter(t *testing.T) {
	root := t.TempDir()
	w, err := newTUIWatcher(root)
	if err != nil {
		t.Fatal(err)
	}
	defer w.close()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("note"), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-w.messages:
		if _, ok := msg.(tuiChanged); !ok {
			t.Fatalf("unexpected message: %#v", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no refresh message")
	}
}
