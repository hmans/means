package collection

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrSchemaIncompatible = errors.New("proposed schema has document violations; use --force to install it anyway")

// SchemaSetResult reports the document check, including when installation is refused.
type SchemaSetResult struct {
	Installed bool `json:"installed"`
	*DoctorResult
}

// SetSchema validates and installs a schema in an existing collection. The old
// schema may be invalid. Force bypasses document diagnostics only. Checks are
// optimistic: concurrent edits can race validation and the final replacement.
func SetSchema(path string, schema []byte, force bool) (*SchemaSetResult, error) {
	c, err := compile(path, schema)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	stat, err := root.Lstat(SchemaFilename)
	if err != nil {
		return nil, fmt.Errorf("inspect existing schema (use init for a new collection): %w", err)
	}
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file, not a symlink or directory", SchemaFilename)
	}
	previous, err := root.ReadFile(SchemaFilename)
	if err != nil {
		return nil, err
	}
	report, err := c.Doctor()
	if err != nil {
		return nil, err
	}
	result := &SchemaSetResult{DoctorResult: report}
	if !report.Healthy && !force {
		return result, ErrSchemaIncompatible
	}
	temp, err := os.CreateTemp(c.Path, ".means-schema-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create schema temporary file: %w", err)
	}
	name := filepath.Base(temp.Name())
	defer root.Remove(name)
	_, writeErr := temp.Write(schema)
	if writeErr == nil {
		writeErr = temp.Chmod(stat.Mode().Perm())
	}
	if writeErr == nil {
		writeErr = temp.Sync()
	}
	if err := errors.Join(writeErr, temp.Close()); err != nil {
		return nil, fmt.Errorf("write schema: %w", err)
	}
	latestStat, err := root.Lstat(SchemaFilename)
	if err != nil {
		return nil, err
	}
	if !latestStat.Mode().IsRegular() || !os.SameFile(stat, latestStat) {
		return nil, fmt.Errorf("%w: schema changed during validation", ErrConflict)
	}
	latest, err := root.ReadFile(SchemaFilename)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(previous, latest) {
		return nil, fmt.Errorf("%w: schema changed during validation", ErrConflict)
	}
	if err := root.Rename(name, SchemaFilename); err != nil {
		return nil, fmt.Errorf("replace schema: %w", err)
	}
	result.Installed = true
	return result, nil
}
