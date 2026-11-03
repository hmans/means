package collection

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const watchDelay = 200 * time.Millisecond

// WatchEvent invalidates a collection snapshot. Reload configuration and documents
// after each event, including events with an error.
type WatchEvent struct {
	// Err reports a watcher failure. Events can have been lost. The watcher
	// attempts to reconcile directory watches and continues until cancellation.
	Err error
}

type collectionWatcher struct {
	w        *fsnotify.Watcher
	root     string
	messages chan WatchEvent
	ctx      context.Context
}

// Watch monitors a collection directory until ctx is canceled. The directory
// must exist, but MEANS.md need not be valid or present. Watch does not load
// configuration, change files, or contact embedding services.
//
// Start Watch before the initial read. No initial event is sent. Changes are
// coalesced after 200ms of quiet; events are invalidations, not an operation log.
// Hidden paths and symlinks below the root are ignored. Nested collections are
// excluded, but adding or removing their MEANS.md boundary invalidates the parent.
// Extra invalidations are possible. Delivery depends on filesystem notification
// support; this is not a durable or lossless subscription.
//
// Cancel ctx when finished. The channel closes after watcher resources are
// released, even if the consumer stops receiving. The root's parent must remain
// present; removal and recreation of the root itself are supported. A symlink
// supplied as the root is resolved once at startup.
func Watch(ctx context.Context, root string) (<-chan WatchEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "watch", Path: root, Err: errors.New("not a directory")}
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	v := &collectionWatcher{w: w, root: root, messages: make(chan WatchEvent), ctx: ctx}
	// The parent watch also catches removal/replacement of the root directory.
	if err = w.Add(filepath.Dir(root)); err == nil {
		err = v.syncDirectories()
	}
	if err != nil {
		w.Close()
		return nil, err
	}
	go v.run()
	return v.messages, nil
}

// Watch directories rather than files so atomic editor replacements are caught.
// Watch a nested collection's directory only to detect its boundary disappearing.
func (v *collectionWatcher) syncDirectories() error {
	wanted := map[string]bool{filepath.Dir(v.root): true}
	err := filepath.WalkDir(v.root, func(path string, d fs.DirEntry, err error) error {
		if ctxErr := v.ctx.Err(); ctxErr != nil {
			return ctxErr
		}
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

func (v *collectionWatcher) relevant(name string) bool {
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

func (v *collectionWatcher) run() {
	defer close(v.messages)
	defer v.w.Close()
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()
	var pending WatchEvent
	var ready bool
	for {
		var out chan WatchEvent
		if ready {
			out = v.messages
		}
		select {
		case <-v.ctx.Done():
			return
		case event, ok := <-v.w.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 && v.relevant(event.Name) {
				timer.Reset(watchDelay)
			}
		case err, ok := <-v.w.Errors:
			if !ok {
				return
			}
			pending.Err = err
			ready = true
			// Reconcile after overflow as events may have been dropped.
			timer.Reset(watchDelay)
		case <-timer.C:
			if err := v.syncDirectories(); err != nil {
				pending.Err = err
				timer.Reset(watchDelay)
			}
			ready = true
		case out <- pending:
			pending = WatchEvent{}
			ready = false
		}
	}
}
