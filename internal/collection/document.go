package collection

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var documentID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

type Document struct {
	ID          string         `json:"id"`
	Hash        string         `json:"hash"`
	Frontmatter map[string]any `json:"frontmatter"`
	Body        string         `json:"body"`
	Markdown    string         `json:"-"`
}

type DocumentSummary struct {
	ID          string         `json:"id"`
	Frontmatter map[string]any `json:"frontmatter"`
	Error       string         `json:"error,omitempty"`
}

type ListResult struct {
	Documents []DocumentSummary `json:"documents"`
	HasMore   bool              `json:"has_more"`
}

// Create validates metadata, assigns an ID, and writes a new Markdown file.
func (c *Collection) Create(frontmatter map[string]any, body string) (*Document, error) {
	if frontmatter == nil {
		return nil, fmt.Errorf("frontmatter must be an object; use {} for empty metadata")
	}
	// Normalize Go caller values into the same JSON types as CLI input.
	encoded, err := json.Marshal(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("encode frontmatter: %w", err)
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	frontmatter = value.(map[string]any)
	if err := c.Validate(frontmatter); err != nil {
		return nil, err
	}
	markdown, err := encodeMarkdown(frontmatter, body)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	for {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, fmt.Errorf("generate document ID: %w", err)
		}
		id := hex.EncodeToString(random[:])
		filename := id + ".md"
		file, err := root.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("create document: %w", err)
		}
		_, writeErr := file.WriteString(markdown)
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return nil, fmt.Errorf("write document: %w", errors.Join(err, root.Remove(filename)))
		}
		return &Document{ID: id, Hash: contentHash(markdown), Frontmatter: frontmatter, Body: body, Markdown: markdown}, nil
	}
}

// Read parses stored content without enforcing the current frontmatter schema,
// allowing consumers to retrieve documents that need migration or repair.
func (c *Collection) Read(id string) (*Document, error) {
	if err := validateDocumentID(id); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return readDocument(root, id)
}

func validateDocumentID(id string) error {
	if id+".md" == InstructionsFilename {
		return fmt.Errorf("%s contains collection instructions, not a document; read the file directly or use MCP info", InstructionsFilename)
	}
	if !documentID.MatchString(id) {
		return fmt.Errorf("invalid document ID %q; use the filename stem (letters, digits, - and _, starting with a letter or digit)", id)
	}
	return nil
}

func contentHash(markdown string) string {
	sum := sha256.Sum256([]byte(markdown))
	return hex.EncodeToString(sum[:])
}

func readDocument(root *os.Root, id string) (*Document, error) {
	filename := id + ".md"
	info, err := root.Lstat(filename)
	if err != nil {
		return nil, fmt.Errorf("read document %q: %w", id, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("document %q must be a regular file, not a symlink or directory", id)
	}
	data, err := root.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read document %q: %w", id, err)
	}
	frontmatter, body, err := parseMarkdown(data)
	if err != nil {
		return nil, fmt.Errorf("document %q: %w", id, err)
	}
	return &Document{ID: id, Hash: contentHash(string(data)), Frontmatter: frontmatter, Body: body, Markdown: string(data)}, nil
}

// List returns metadata in ascending filename order. Zero means no limit.
// Malformed documents occupy a result entry with an error instead of metadata.
func (c *Collection) List(limit int) (*ListResult, error) {
	if limit < 0 {
		return nil, fmt.Errorf("limit must be zero or greater")
	}
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	result := &ListResult{Documents: []DocumentSummary{}}
	for _, entry := range entries {
		if entry.Name() == InstructionsFilename {
			continue
		}
		id, markdown := strings.CutSuffix(entry.Name(), ".md")
		if !markdown || !documentID.MatchString(id) || !entry.Type().IsRegular() {
			continue
		}
		if limit > 0 && len(result.Documents) == limit {
			result.HasMore = true
			break
		}
		doc, err := readDocument(root, id)
		summary := DocumentSummary{ID: id}
		if err != nil {
			summary.Error = err.Error()
		} else {
			summary.Frontmatter = doc.Frontmatter
		}
		result.Documents = append(result.Documents, summary)
	}
	return result, nil
}
