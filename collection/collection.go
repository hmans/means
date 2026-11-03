package collection

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ConfigFilename is the collection's configuration and workflow file.
const ConfigFilename = "MEANS.md"

//go:embed default.schema.json
var defaultSchema []byte

// Collection is a loaded collection with a compiled application data schema.
// Use Load, Init, or InitConfig; its zero value is not usable. Configuration is
// a fixed snapshot while document operations scan current files. No Close is
// needed. Its accessors return copies, and it is safe for concurrent operations.
type Collection struct {
	path              string
	app               AppConfig       // Optional views from the same MEANS.md snapshot.
	schema            json.RawMessage // The JSON schema compiled for this collection.
	validator         *jsonschema.Schema
	config            map[string]any
	body              string
	configHash        string
	ctx               context.Context
	embeddingProgress func(string)
	embeddings        *EmbeddingConfig
}

// Path returns the absolute collection directory.
func (c *Collection) Path() string { return c.path }

// Schema returns a copy of the JSON Schema from the loaded snapshot.
func (c *Collection) Schema() json.RawMessage { return slices.Clone(c.schema) }

// App returns a copy of the optional application views from the loaded snapshot.
func (c *Collection) App() AppConfig {
	app := c.app
	app.Views = slices.Clone(app.Views)
	for i := range app.Views {
		app.Views[i].Columns = slices.Clone(app.Views[i].Columns)
	}
	return app
}

// WithEmbeddingProgress returns a copy that reports embedding work to progress.
// The collection package stays silent unless a caller supplies this callback.
// Nil disables progress reporting. Concurrent operations can invoke the callback
// concurrently; the callback must synchronize its own shared state.
func (c *Collection) WithEmbeddingProgress(progress func(string)) *Collection {
	copy := *c
	copy.embeddingProgress = progress
	return &copy
}

// operation gives each call its own context without changing the loaded snapshot.
func (c *Collection) operation(ctx context.Context) (*Collection, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	copy := *c
	copy.ctx = ctx
	return &copy, nil
}

// ErrValidation identifies an invalid document envelope or application data.
// Validate and ValidateData wrap it, as do write failures during schema/envelope
// validation. The underlying validation error remains available through
// errors.As. Other invalid arguments or source-parsing failures can return
// errors without this sentinel.
var ErrValidation = errors.New("document validation failed")

// Validate checks the means envelope and data against the embedded schema.
// Frontmatter must contain only a valid id, a nonempty title, and a data object.
// JSON-compatible Go values are normalized without changing the map. It does not
// read or write documents; failures wrap ErrValidation.
func (c *Collection) Validate(frontmatter map[string]any) error {
	normalized, err := normalizeObject(frontmatter)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrValidation, err)
	}
	return c.validate(normalized)
}

// Internal callers already hold normalized document values.
func (c *Collection) validate(frontmatter map[string]any) error {
	if err := validateEnvelope(frontmatter); err != nil {
		return fmt.Errorf("%w: %w", ErrValidation, err)
	}
	return c.validateData(frontmatter["data"].(map[string]any))
}

// ValidateData checks application data independently of the means-owned fields.
// Supply the application object itself, not a frontmatter envelope. Values are
// normalized like CreateOptions.Data; nil is not an object. It uses the loaded
// schema without reading or writing documents. Failures wrap ErrValidation.
func (c *Collection) ValidateData(data map[string]any) error {
	normalized, err := normalizeObject(data)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrValidation, err)
	}
	return c.validateData(normalized)
}

func (c *Collection) validateData(data map[string]any) error {
	if err := c.validator.Validate(data); err != nil {
		return fmt.Errorf("%w: frontmatter.data does not satisfy the schema: %w", ErrValidation, err)
	}
	return nil
}

// Init validates a schema, then creates the directory and writes MEANS.md
// without overwriting an existing file. A nil schema selects the example schema.
// Schema is a JSON Schema object or boolean in JSON bytes, not YAML. Schemas are
// self-contained; external references are rejected. Draft 2020-12 is the default.
// The example schema allows only optional tags (an array of strings). Init adds
// a .means/ rule to .gitignore while preserving existing content. Use InitConfig
// to supply identity and workflow instructions with the schema.
func Init(path string, schema []byte) (*Collection, error) {
	if schema == nil {
		schema = defaultSchema
	}
	c, err := compile(path, schema)
	if err != nil {
		return nil, err
	}
	value, err := orderedSchema(schema)
	if err != nil {
		return nil, err
	}
	content, err := encodeConfig(map[string]any{"title": filepath.Base(c.path), "schema": value}, "# Collection guidance\n\nDescribe this collection and its workflow here.\n")
	if err != nil {
		return nil, err
	}
	return InitConfig(path, []byte(content))
}

// InitConfig validates and installs a complete MEANS.md without overwriting it.
// Content is YAML frontmatter (including schema), followed by optional Markdown
// instructions. Exact bytes are preserved. Validation occurs before creating
// files; directory creation and the .gitignore rule follow the Init contract.
// For example, a minimal content value is:
//
//	---
//	title: Notes
//	schema: true
//	---
//	Review notes before publishing.
func InitConfig(path string, content []byte) (*Collection, error) {
	c, err := parseConfig(path, content)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.path, 0o755); err != nil {
		return nil, fmt.Errorf("create collection directory: %w", err)
	}
	ignorePath := filepath.Join(c.path, ".gitignore")
	var ignore []byte
	ignoreMode := os.FileMode(0o644)
	if stat, err := os.Lstat(ignorePath); err == nil {
		if !stat.Mode().IsRegular() {
			return nil, fmt.Errorf(".gitignore must be a regular file, not a symlink or directory")
		}
		ignoreMode = stat.Mode().Perm()
		ignore, err = os.ReadFile(ignorePath)
		if err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	hasRule := false
	for _, line := range strings.Split(string(ignore), "\n") {
		if strings.TrimSpace(line) == ".means/" || strings.TrimSpace(line) == "/.means/" {
			hasRule = true
		}
	}
	filename := filepath.Join(c.path, ConfigFilename)
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("%s already exists; refusing to overwrite it", filename)
	}
	if err != nil {
		return nil, fmt.Errorf("create collection configuration: %w", err)
	}
	_, writeErr := f.Write(content)
	if err := errors.Join(writeErr, f.Close()); err != nil {
		removeErr := os.Remove(filename)
		return nil, fmt.Errorf("write collection configuration: %w", errors.Join(err, removeErr))
	}
	if !hasRule {
		if len(ignore) > 0 && ignore[len(ignore)-1] != '\n' {
			ignore = append(ignore, '\n')
		}
		ignore = append(ignore, []byte(".means/\n")...)
		temp, err := os.CreateTemp(c.path, ".means-gitignore-*.tmp")
		if err != nil {
			return nil, errors.Join(err, os.Remove(filename))
		}
		defer os.Remove(temp.Name())
		_, writeErr := temp.Write(ignore)
		if writeErr == nil {
			writeErr = temp.Chmod(ignoreMode)
		}
		if writeErr == nil {
			writeErr = temp.Sync()
		}
		if err := errors.Join(writeErr, temp.Close()); err != nil {
			return nil, errors.Join(err, os.Remove(filename))
		}
		if err := os.Rename(temp.Name(), ignorePath); err != nil {
			return nil, errors.Join(err, os.Remove(filename))
		}
	}
	return c, nil
}

// Load reads the exact directory supplied; it does not search parent directories.
// The returned configuration is a snapshot. Reload after external config edits.
// It compiles the schema but does not validate existing documents or create files.
// Missing MEANS.md wraps fs.ErrNotExist. Document operations do not automatically
// reload this snapshot after SetSchema or SetInstructions; call Load again and
// reapply any host embedding settings and progress callback.
func Load(path string) (*Collection, error) {
	filename := filepath.Join(path, ConfigFilename)
	stat, err := os.Lstat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no collection configuration at %s: %w", filename, err)
	}
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", filename, err)
	}
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file, not a symlink or directory", filename)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filename, err)
	}
	c, err := parseConfig(path, data)
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
	location := filepath.Join(path, ConfigFilename)
	if err := compiler.AddResource(location, doc); err != nil {
		return nil, fmt.Errorf("register JSON Schema: %w", err)
	}
	validator, err := compiler.Compile(location)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON Schema: %w", err)
	}
	return &Collection{path: path, schema: append(json.RawMessage(nil), schema...), validator: validator}, nil
}

// Schemas are self-contained for now. The compiler still resolves internal
// references and its bundled standard metaschemas without this loader.
type selfContainedLoader struct{}

func (selfContainedLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external schema reference %q is not supported; use internal references such as #/$defs/name", url)
}
