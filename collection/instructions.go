package collection

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// SetInstructions replaces only the MEANS.md body. The hash covers the complete
// file, including its frontmatter. External writes can still race replacement.
func (c *Collection) SetInstructions(ctx context.Context, hash, instructions string) (*Description, error) {
	c, err := c.operation(ctx)
	if err != nil {
		return nil, err
	}
	if !validHash.MatchString(hash) {
		return nil, fmt.Errorf("hash is required and must be 64 lowercase hexadecimal SHA-256 characters; use the hash from Describe")
	}
	if !utf8.ValidString(instructions) {
		return nil, fmt.Errorf("instructions must be valid UTF-8")
	}
	root, err := os.OpenRoot(c.path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	stat, err := root.Lstat(ConfigFilename)
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file, not a symlink or directory", ConfigFilename)
	}
	previous, err := root.ReadFile(ConfigFilename)
	if err != nil {
		return nil, err
	}
	if contentHash(string(previous)) != hash {
		return nil, fmt.Errorf("%w: MEANS.md changed; reload and reconcile the instructions before retrying", ErrConflict)
	}
	current, err := parseConfig(c.path, previous)
	if err != nil {
		return nil, err
	}
	prefix := string(previous[:len(previous)-len(current.body)])
	if instructions != "" && !strings.HasSuffix(prefix, "\n") {
		prefix += "\n"
	}
	content := prefix + instructions
	if err := replaceConfig(root, c.path, stat, previous, content); err != nil {
		return nil, err
	}
	current.body, current.configHash = instructions, contentHash(content)
	return current.Describe(), nil
}
