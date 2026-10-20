package collection

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// InstructionsFilename is collection configuration, not a stored document.
const InstructionsFilename = "instructions.md"

// Description gives consumers the collection's purpose and author-supplied workflow.
type Description struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
}

// Describe reads top-level schema annotations and optional instructions.md.
// Missing titles fall back to the collection directory name, including for boolean schemas.
func (c *Collection) Describe() (*Description, error) {
	info := &Description{Title: filepath.Base(c.Path)}
	var annotations struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	// A schema may be a boolean rather than an object.
	if strings.HasPrefix(strings.TrimSpace(string(c.Schema)), "{") {
		if err := json.Unmarshal(c.Schema, &annotations); err != nil {
			return nil, fmt.Errorf("read schema annotations: %w", err)
		}
		if title := strings.TrimSpace(annotations.Title); title != "" {
			info.Title = title
		}
		info.Description = strings.TrimSpace(annotations.Description)
	}
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	stat, err := root.Lstat(InstructionsFilename)
	if errors.Is(err, os.ErrNotExist) {
		return info, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", InstructionsFilename, err)
	}
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file, not a symlink or directory", InstructionsFilename)
	}
	data, err := root.ReadFile(InstructionsFilename)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", InstructionsFilename, err)
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%s must contain valid UTF-8", InstructionsFilename)
	}
	info.Instructions = string(data)
	return info, nil
}
