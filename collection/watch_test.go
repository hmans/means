package collection

import (
	"context"
	"errors"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatch(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, err := Watch(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	changed := func() {
		t.Helper()
		select {
		case msg := <-events:
			if msg.Err != nil {
				t.Fatalf("unexpected watch message: %#v", msg)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("filesystem change was not reported")
		}
	}
	quiet := func() {
		t.Helper()
		select {
		case msg := <-events:
			t.Fatalf("unexpected notification: %#v", msg)
		case <-time.After(2 * watchDelay):
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

func TestWatchCancellationWithoutReader(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	events, err := Watch(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	// Let a notification become pending while no consumer receives it.
	time.Sleep(2 * watchDelay)
	cancel()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case _, ok := <-events:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("cancellation did not close the channel")
		}
	}
}

func TestWatchInvalidStart(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Watch(ctx, root); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if _, err := Watch(context.Background(), filepath.Join(root, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Watch(context.Background(), file); err == nil {
		t.Fatal("accepted regular file")
	}
}

func watchChange(t *testing.T, events <-chan WatchEvent) WatchEvent {
	t.Helper()
	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("watch stopped early")
		}
		return event
	case <-time.After(3 * time.Second):
		t.Fatal("no watch notification")
		return WatchEvent{}
	}
}

func TestWatchRootReplacement(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "collection")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, err := Watch(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if event := watchChange(t, events); event.Err != nil {
		t.Fatal(event.Err)
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if event := watchChange(t, events); event.Err != nil {
		t.Fatal(event.Err)
	}
	if err := os.WriteFile(filepath.Join(root, "MEANS.md"), []byte("invalid config still invalidates"), 0600); err != nil {
		t.Fatal(err)
	}
	if event := watchChange(t, events); event.Err != nil {
		t.Fatal(event.Err)
	}
}

func TestWatchOverflowReconcilesDirectories(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	backend, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	v := &collectionWatcher{w: backend, root: root, ctx: ctx, messages: make(chan WatchEvent)}
	// No initial watches: simulate a directory creation lost during overflow.
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	go v.run()
	select {
	case backend.Errors <- fsnotify.ErrEventOverflow:
	case <-time.After(3 * time.Second):
		t.Fatal("watch did not receive error")
	}
	if event := watchChange(t, v.messages); !errors.Is(event.Err, fsnotify.ErrEventOverflow) {
		t.Fatalf("got %v", event.Err)
	}
	if event := watchChange(t, v.messages); event.Err != nil {
		t.Fatal(event.Err)
	}
	if err := os.WriteFile(filepath.Join(child, "note.md"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if event := watchChange(t, v.messages); event.Err != nil {
		t.Fatal(event.Err)
	}
	cancel()
	for range v.messages {
	}
}
