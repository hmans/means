// Package collection manages Markdown collections and their application data schemas.
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
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const ConfigFilename = "MEANS.md"

//go:embed default.schema.json
var defaultSchema []byte

// Collection is a loaded collection with a compiled application data schema.
type Collection struct {
	Path              string
	Schema            json.RawMessage // The JSON schema compiled for this collection.
	validator         *jsonschema.Schema
	config            map[string]any
	body              string
	ctx               context.Context
	embeddingProgress func(string)
	embeddings        *EmbeddingConfig
}

// WithEmbeddingProgress returns a copy that reports embedding work to progress.
// The collection package stays silent unless a caller supplies this callback.
func (c *Collection) WithEmbeddingProgress(progress func(string)) *Collection {
	copy := *c
	copy.embeddingProgress = progress
	return &copy
}

// WithContext returns a copy whose embedding requests and lock waits can be
// canceled with ctx. A nil context uses context.Background().
func (c *Collection) WithContext(ctx context.Context) *Collection {
	copy := *c
	copy.ctx = ctx
	return &copy
}

// Validate checks the means envelope and data against the embedded schema.
func (c *Collection) Validate(frontmatter map[string]any) error {
	if err := validateEnvelope(frontmatter); err != nil {
		return err
	}
	return c.ValidateData(frontmatter["data"].(map[string]any))
}

// ValidateData checks application data independently of the means-owned fields.
func (c *Collection) ValidateData(data map[string]any) error {
	if err := c.validator.Validate(data); err != nil {
		return fmt.Errorf("frontmatter.data does not satisfy the schema: %w", err)
	}
	return nil
}

// Init validates a schema, then creates the directory and writes MEANS.md
// without overwriting an existing file. A nil schema selects the example schema.
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
	content, err := encodeConfig(map[string]any{"title": filepath.Base(c.Path), "schema": value}, "# Collection guidance\n\nDescribe this collection and its workflow here.\n")
	if err != nil {
		return nil, err
	}
	return InitConfig(path, []byte(content))
}

// InitConfig validates and installs a complete MEANS.md without overwriting it.
func InitConfig(path string, content []byte) (*Collection, error) {
	c, err := parseConfig(path, content)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.Path, 0o755); err != nil {
		return nil, fmt.Errorf("create collection directory: %w", err)
	}
	ignorePath := filepath.Join(c.Path, ".gitignore")
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
	filename := filepath.Join(c.Path, ConfigFilename)
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
		temp, err := os.CreateTemp(c.Path, ".means-gitignore-*.tmp")
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
func Load(path string) (*Collection, error) {
	filename := filepath.Join(path, ConfigFilename)
	stat, err := os.Lstat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no collection configuration at %s; run means init --path %q", filename, path)
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
	return &Collection{Path: path, Schema: append(json.RawMessage(nil), schema...), validator: validator}, nil
}

// Schemas are self-contained for now. The compiler still resolves internal
// references and its bundled standard metaschemas without this loader.
type selfContainedLoader struct{}

func (selfContainedLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external schema reference %q is not supported; use internal references such as #/$defs/name", url)
}
