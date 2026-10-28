package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/fsnotify/fsnotify"
)

const tuiWatchDelay = 200 * time.Millisecond

type tuiWatchError struct{ err error }

type tuiWatcher struct {
	w        *fsnotify.Watcher
	root     string
	messages chan tea.Msg
	done     chan struct{}
	stopped  chan struct{}
}

func newTUIWatcher(root string) (*tuiWatcher, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	v := &tuiWatcher{w: w, root: root, messages: make(chan tea.Msg), done: make(chan struct{}), stopped: make(chan struct{})}
	// The parent watch also catches removal/replacement of the root directory.
	if err = w.Add(filepath.Dir(root)); err == nil {
		err = v.syncDirectories()
	}
	if err != nil {
		w.Close()
		return nil, err
	}
	go v.run()
	return v, nil
}

func (v *tuiWatcher) close() {
	close(v.done)
	v.w.Close()
	<-v.stopped
}

// Watch directories rather than files so atomic editor replacements are caught.
// Watch a nested collection's directory only to detect its boundary disappearing.
func (v *tuiWatcher) syncDirectories() error {
	wanted := map[string]bool{filepath.Dir(v.root): true}
	err := filepath.WalkDir(v.root, func(path string, d fs.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != v.root && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
		wanted[path] = true
		if err := v.w.Add(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if path != v.root {
			if _, err := os.Lstat(filepath.Join(path, "MEANS.md")); err == nil {
				return fs.SkipDir
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, path := range v.w.WatchList() {
		if !wanted[path] {
			_ = v.w.Remove(path)
		}
	}
	return nil
}

func (v *tuiWatcher) relevant(name string) bool {
	rel, err := filepath.Rel(v.root, name)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	if rel == "." {
		return true
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if strings.HasPrefix(part, ".") {
			return false
		}
	}
	parent := filepath.Dir(name)
	if parent != v.root && filepath.Base(name) != "MEANS.md" {
		if _, err := os.Lstat(filepath.Join(parent, "MEANS.md")); err == nil {
			return false
		}
	}
	return true
}

func (v *tuiWatcher) run() {
	defer close(v.stopped)
	defer close(v.messages)
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()
	var pending tea.Msg
	for {
		var out chan tea.Msg
		if pending != nil {
			out = v.messages
		}
		select {
		case <-v.done:
			return
		case event, ok := <-v.w.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 && v.relevant(event.Name) {
				timer.Reset(tuiWatchDelay)
			}
		case err, ok := <-v.w.Errors:
			if !ok {
				return
			}
			pending = tuiWatchError{err}
			// Reconcile after overflow as events may have been dropped.
			timer.Reset(tuiWatchDelay)
		case <-timer.C:
			if err := v.syncDirectories(); err != nil {
				pending = tuiWatchError{err}
			} else {
				pending = tuiChanged{}
			}
		case out <- pending:
			pending = nil
		}
	}
}
