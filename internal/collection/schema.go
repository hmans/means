package collection

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	stat, err := root.Lstat(ConfigFilename)
	if err != nil {
		return nil, fmt.Errorf("inspect existing schema (use init for a new collection): %w", err)
	}
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file, not a symlink or directory", ConfigFilename)
	}
	previous, err := root.ReadFile(ConfigFilename)
	if err != nil {
		return nil, err
	}
	fields, body, _, err := configFields(previous)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ConfigFilename, err)
	}
	if fields["schema"], err = orderedSchema(schema); err != nil {
		return nil, err
	}
	content, err := encodeConfig(fields, body)
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
	if err := replaceConfig(root, c.Path, stat, previous, content); err != nil {
		return nil, err
	}
	result.Installed = true
	return result, nil
}

// orderedSchema re-encodes JSON token by token, keeping key order and exact
// numbers. Re-encoding strings normalizes escapes such as \/, which are valid
// JSON but rejected by the YAML parser.
func orderedSchema(schema []byte) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(schema))
	decoder.UseNumber()
	type container struct {
		object bool
		items  int
	}
	var out bytes.Buffer
	var stack []container
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return out.Bytes(), nil
		}
		if err != nil {
			return nil, fmt.Errorf("parse JSON Schema: %w", err)
		}
		if delim, ok := token.(json.Delim); ok && (delim == '}' || delim == ']') {
			stack = stack[:len(stack)-1]
			out.WriteRune(rune(delim))
			continue
		}
		if len(stack) > 0 {
			top := &stack[len(stack)-1]
			if top.items > 0 && top.object && top.items%2 == 1 {
				out.WriteByte(':')
			} else if top.items > 0 {
				out.WriteByte(',')
			}
			top.items++
		}
		switch value := token.(type) {
		case json.Delim:
			out.WriteRune(rune(value))
			stack = append(stack, container{object: value == '{'})
		case json.Number:
			out.WriteString(value.String())
		default:
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			out.Write(encoded)
		}
	}
}

// replaceConfig rejects observed changes to any part of the configuration.
// A concurrent write can still race the final rename; this is not a lock.
func replaceConfig(root *os.Root, directory string, previousInfo os.FileInfo, previous []byte, content string) error {
	temp, err := os.CreateTemp(directory, ".means-schema-*.tmp")
	if err != nil {
		return fmt.Errorf("create schema temporary file: %w", err)
	}
	name := filepath.Base(temp.Name())
	defer root.Remove(name)
	_, writeErr := temp.WriteString(content)
	if writeErr == nil {
		writeErr = temp.Chmod(previousInfo.Mode().Perm())
	}
	if writeErr == nil {
		writeErr = temp.Sync()
	}
	if err := errors.Join(writeErr, temp.Close()); err != nil {
		return fmt.Errorf("write schema: %w", err)
	}
	latestStat, err := root.Lstat(ConfigFilename)
	if err != nil {
		return err
	}
	if !latestStat.Mode().IsRegular() || !os.SameFile(previousInfo, latestStat) {
		return fmt.Errorf("%w: configuration changed during validation", ErrConflict)
	}
	latest, err := root.ReadFile(ConfigFilename)
	if err != nil {
		return err
	}
	if !bytes.Equal(previous, latest) {
		return fmt.Errorf("%w: configuration changed during validation", ErrConflict)
	}
	if err := root.Rename(name, ConfigFilename); err != nil {
		return fmt.Errorf("replace schema: %w", err)
	}
	return nil
}
