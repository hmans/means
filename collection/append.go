package collection

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Append adds text to the end of a document's Markdown body without inserting
// separators or newlines. An empty hash appends to the latest stored version;
// otherwise use the current hash from Read. Empty text leaves the body unchanged.
// The result must satisfy the loaded schema. Append and Set serialize on a
// collection-local edit lock with a context-aware wait of up to five seconds.
// Like a body-only Update, Append preserves frontmatter bytes, filename, and
// permissions, maintains enabled embeddings before saving, and rejects observed
// changes with ErrConflict. External writes can still race the final replacement.
// The returned Document contains the new content hash.
func (c *Collection) Append(ctx context.Context, id, hash, text string) (*Document, error) {
	return c.modify(ctx, id, hash, func(current *Document) (Patch, bool, error) {
		body := current.Body + text
		return Patch{Body: &body}, false, nil
	})
}

// modify reads the latest document under the edit lock, checks an optional hash,
// and applies the patch built from it. replace selects whole-metadata replacement.
func (c *Collection) modify(ctx context.Context, id, hash string, build func(*Document) (Patch, bool, error)) (*Document, error) {
	c, err := c.operation(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateDocumentID(id); err != nil {
		return nil, err
	}
	if hash != "" && !validHash.MatchString(hash) {
		return nil, fmt.Errorf("hash must be 64 lowercase hexadecimal SHA-256 characters")
	}
	cache := filepath.Join(c.path, ".means")
	if stat, err := os.Lstat(cache); err == nil && !stat.IsDir() {
		return nil, fmt.Errorf(".means must be a directory, not a symlink")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.MkdirAll(cache, 0o700); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(cache, "edit.lock")
	if stat, err := os.Lstat(lockPath); err == nil && !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("edit lock must be a regular file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	lock, err := tryLock(ctx, lockPath, "edits", "edits are busy; retry after the current edit finishes")
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()
	current, err := c.Read(ctx, id)
	if err != nil {
		return nil, err
	}
	if hash != "" && current.Hash != hash {
		return nil, conflict(id)
	}
	patch, replace, err := build(current)
	if err != nil {
		return nil, err
	}
	return c.update(id, current.Hash, patch, replace)
}
