package collection

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	ErrConflict = errors.New("document content has changed")
	validHash   = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Patch merges supplied metadata recursively (null deletes, arrays replace).
// A nil Frontmatter or Body leaves that portion of the document unchanged.
type Patch struct {
	Frontmatter map[string]any
	Body        *string
}

// Update checks the exact stored bytes against hash before applying a patch.
// Replacement uses a temporary file in the same directory. Hash checks are
// optimistic: a simultaneous write can still race the final replacement.
func (c *Collection) Update(id, hash string, patch Patch) (*Document, error) {
	return c.update(id, hash, patch, false)
}

// ReplaceMarkdown validates an edited document and replaces its content using
// the same hash checks and embedding maintenance as Update. Metadata is replaced,
// not merge-patched, so removed keys and literal null values are preserved.
func (c *Collection) ReplaceMarkdown(id, hash string, markdown []byte) (*Document, error) {
	doc, err := parseDocumentFile("edited document", markdown, "")
	if err != nil {
		return nil, err
	}
	if doc.ID != id {
		return nil, fmt.Errorf("id is immutable; retain %q", id)
	}
	delete(doc.Frontmatter, "id")
	return c.update(id, hash, Patch{Frontmatter: doc.Frontmatter, Body: &doc.Body}, true)
}

func (c *Collection) update(id, hash string, patch Patch, replace bool) (*Document, error) {
	if err := validateDocumentID(id); err != nil {
		return nil, err
	}
	if !validHash.MatchString(hash) {
		return nil, fmt.Errorf("hash is required and must be 64 lowercase hexadecimal SHA-256 characters; read the document first")
	}
	if patch.Frontmatter == nil && patch.Body == nil {
		return nil, fmt.Errorf("supply frontmatter and/or body to update")
	}
	if _, supplied := patch.Frontmatter["id"]; supplied {
		return nil, fmt.Errorf("id is immutable; omit it from the frontmatter patch")
	}
	for key := range patch.Frontmatter {
		if key != "title" && key != "data" {
			return nil, fmt.Errorf("unknown means frontmatter field %q; application fields belong under data", key)
		}
	}
	if patch.Frontmatter != nil {
		var err error
		patch.Frontmatter, err = normalizeObject(patch.Frontmatter)
		if err != nil {
			return nil, err
		}
	}
	embeddings, err := c.openEmbeddings()
	if err != nil {
		return nil, err
	}
	defer embeddings.close()
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	current, err := readDocument(root, id)
	if err != nil {
		return nil, err
	}
	if current.Hash != hash {
		return nil, conflict(id)
	}
	frontmatter := current.Frontmatter
	if patch.Frontmatter != nil {
		if replace {
			frontmatter = patch.Frontmatter
			frontmatter["id"] = id
		} else {
			frontmatter = mergeObject(frontmatter, patch.Frontmatter)
		}
	}
	if err := c.Validate(frontmatter); err != nil {
		return nil, err
	}
	body := current.Body
	if patch.Body != nil {
		body = *patch.Body
	}
	if !utf8.ValidString(body) {
		return nil, fmt.Errorf("body must be valid UTF-8")
	}
	if err := embeddings.upsert(id, frontmatter, body); err != nil {
		return nil, err
	}
	if err := embeddings.save(); err != nil {
		return nil, err
	}
	metadataChanged := !reflect.DeepEqual(frontmatter, current.Frontmatter)
	if !metadataChanged && body == current.Body {
		if embeddings != nil {
			latest, err := readDocument(root, id)
			if err != nil {
				return nil, err
			}
			if latest.Hash != hash || latest.filename != current.filename {
				return nil, conflict(id)
			}
		}
		return current, nil
	}
	var markdown string
	if metadataChanged {
		markdown, err = encodeMarkdown(frontmatter, body)
		if err != nil {
			return nil, err
		}
	} else {
		prefix := current.Markdown[:len(current.Markdown)-len(current.Body)]
		if body != "" && !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}
		markdown = prefix + body
	}
	fileInfo, err := root.Lstat(current.filename)
	if err != nil {
		return nil, err
	}
	// Random temporary names are hidden and cannot be document IDs.
	var temp *os.File
	var tempname string
	for {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, err
		}
		tempname = path.Join(path.Dir(current.filename), ".means-"+hex.EncodeToString(random[:])+".tmp")
		temp, err = root.OpenFile(tempname, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if !errors.Is(err, os.ErrExist) {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("create update temporary file: %w", err)
	}
	defer root.Remove(tempname)
	_, writeErr := temp.WriteString(markdown)
	if writeErr == nil {
		writeErr = temp.Chmod(fileInfo.Mode().Perm())
	}
	if writeErr == nil {
		writeErr = temp.Sync()
	}
	if err := errors.Join(writeErr, temp.Close()); err != nil {
		return nil, fmt.Errorf("write update: %w", err)
	}
	// Also catch edits made while preparing the replacement. Another writer
	// can still race the final rename; the hash check and rename are not atomic.
	latest, err := readDocument(root, id)
	if err != nil {
		return nil, err
	}
	if latest.Hash != hash || latest.filename != current.filename {
		return nil, conflict(id)
	}
	if err := root.Rename(tempname, current.filename); err != nil {
		return nil, fmt.Errorf("replace document: %w", err)
	}
	return &Document{ID: id, Hash: contentHash(markdown), Frontmatter: frontmatter, Body: body, Markdown: markdown, filename: current.filename}, nil
}

func conflict(id string) error {
	return fmt.Errorf("%w for %q; read it again, review the changes, and retry with its new hash", ErrConflict, id)
}

// mergeObject implements RFC 7396 for an object patch, without mutating target.
func mergeObject(target, patch map[string]any) map[string]any {
	result := make(map[string]any, len(target))
	for key, value := range target {
		result[key] = value
	}
	for key, value := range patch {
		if value == nil {
			delete(result, key)
			continue
		}
		if object, ok := value.(map[string]any); ok {
			previous, _ := result[key].(map[string]any)
			result[key] = mergeObject(previous, object)
		} else {
			result[key] = value
		}
	}
	return result
}
