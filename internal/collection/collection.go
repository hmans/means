// Package collection manages Markdown collections and their frontmatter schemas.
package collection

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const SchemaFilename = "schema.json"

//go:embed default.schema.json
var defaultSchema []byte

// Collection is a loaded collection with a compiled frontmatter schema.
type Collection struct {
	Path      string
	validator *jsonschema.Schema
}

// Validate checks JSON-compatible frontmatter against the collection's schema.
func (c *Collection) Validate(frontmatter map[string]any) error {
	if err := c.validator.Validate(frontmatter); err != nil {
		return fmt.Errorf("frontmatter does not satisfy the schema: %w", err)
	}
	return nil
}

// Init validates a schema, then creates the directory and writes schema.json
// without overwriting an existing file. A nil schema selects the example schema.
func Init(path string, schema []byte) (*Collection, error) {
	if schema == nil {
		schema = defaultSchema
	}
	c, err := compile(path, schema)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.Path, 0o755); err != nil {
		return nil, fmt.Errorf("create collection directory: %w", err)
	}
	filename := filepath.Join(c.Path, SchemaFilename)
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("%s already exists; refusing to overwrite it", filename)
	}
	if err != nil {
		return nil, fmt.Errorf("create collection schema: %w", err)
	}
	_, writeErr := f.Write(schema)
	if err := errors.Join(writeErr, f.Close()); err != nil {
		removeErr := os.Remove(filename)
		return nil, fmt.Errorf("write collection schema: %w", errors.Join(err, removeErr))
	}
	return c, nil
}

// Load reads the exact directory supplied; it does not search parent directories.
func Load(path string) (*Collection, error) {
	filename := filepath.Join(path, SchemaFilename)
	data, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no collection schema at %s; run means init --path %q", filename, path)
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filename, err)
	}
	c, err := compile(path, data)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", filename, err)
	}
	return c, nil
}

func compile(path string, schema []byte) (*Collection, error) {
	if path == "" {
		return nil, fmt.Errorf("collection path must not be empty")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve collection path: %w", err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return nil, fmt.Errorf("parse JSON Schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(selfContainedLoader{})
	location := filepath.Join(path, SchemaFilename)
	if err := compiler.AddResource(location, doc); err != nil {
		return nil, fmt.Errorf("register JSON Schema: %w", err)
	}
	validator, err := compiler.Compile(location)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON Schema: %w", err)
	}
	return &Collection{Path: path, validator: validator}, nil
}

// Schemas are self-contained for now. The compiler still resolves internal
// references and its bundled standard metaschemas without this loader.
type selfContainedLoader struct{}

func (selfContainedLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external schema reference %q is not supported; use internal references such as #/$defs/name", url)
}
