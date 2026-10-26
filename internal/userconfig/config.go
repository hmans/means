// Package userconfig manages machine-local Means settings. Collection operations
// receive these settings explicitly and never read the user's home directory.
package userconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

type Config struct{ Embeddings *Embeddings }

const Example = `# Means user configuration. Applies to every collection on this machine.
# Uncomment to enable embeddings. Set OPENROUTER_API_KEY in your environment.
# For direct OpenAI use provider: openai and model: text-embedding-3-small,
# and set OPENAI_API_KEY instead. Credentials never belong in this file.
# Restart running MCP servers after changing these settings.
#
# embeddings:
#   provider: openrouter
#   model: openai/text-embedding-3-small
#   dimensions: 512
`

func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate Means user config: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	if !filepath.IsAbs(base) {
		return "", fmt.Errorf("XDG_CONFIG_HOME must be an absolute path")
	}
	return filepath.Join(base, "means", "config.yaml"), nil
}

// Open bootstraps a missing file and loads settings. A bootstrap failure is a
// warning with disabled settings; an existing invalid/unreadable file is an error.
// Callers own reporting, keeping library code and MCP stdout silent.
func Open() (cfg *Config, warning error, err error) {
	filename, err := Path()
	if err != nil {
		return &Config{}, err, nil
	}
	info, err := os.Stat(filename)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		if err := bootstrap(filename); err != nil {
			return &Config{}, fmt.Errorf("could not create Means user config at %s: %w; embeddings disabled", filename, err), nil
		}
	} else if err != nil {
		return nil, nil, fmt.Errorf("inspect Means user config %s: %w", filename, err)
	}
	if info != nil && !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("Means user config %s must be a regular file", filename)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("read Means user config %s: %w", filename, err)
	}
	cfg, err = Parse(data)
	if err != nil {
		return nil, nil, fmt.Errorf("Means user config %s: %w", filename, err)
	}
	return cfg, nil, nil
}

func bootstrap(filename string) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(filename), ".config-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.WriteString(Example)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return err
	}
	// Publish the complete file without replacing an existing file, even when
	// several CLI/MCP processes start together. Temporary files are mode 0600.
	if err := os.Link(f.Name(), filename); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return nil
}

func Parse(data []byte) (*Config, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("configuration must be UTF-8")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	err := decoder.Decode(&doc)
	if err == io.EOF {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var extra yaml.Node
	if decoder.Decode(&extra) != io.EOF {
		return nil, fmt.Errorf("configuration must contain exactly one YAML mapping")
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("configuration must be a YAML mapping")
	}
	root := doc.Content[0]
	if err := plainYAML(root); err != nil {
		return nil, err
	}
	cfg := &Config{}
	seen := map[string]bool{}
	for i := 0; i < len(root.Content); i += 2 {
		key, value := root.Content[i], root.Content[i+1]
		if key.Tag != "!!str" || key.Value != "embeddings" {
			return nil, fmt.Errorf("unknown user configuration field %q", key.Value)
		}
		if seen[key.Value] {
			return nil, fmt.Errorf("duplicate field %q", key.Value)
		}
		seen[key.Value] = true
		if err := plainYAML(value); err != nil {
			return nil, err
		}
		var raw any
		if err := value.Decode(&raw); err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		jsonDecoder := json.NewDecoder(bytes.NewReader(encoded))
		jsonDecoder.UseNumber()
		if err := jsonDecoder.Decode(&raw); err != nil {
			return nil, err
		}
		cfg.Embeddings, err = parseEmbeddings(raw)
		if err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func plainYAML(n *yaml.Node) error {
	if n.Kind == yaml.AliasNode {
		return fmt.Errorf("YAML aliases are not supported")
	}
	switch n.Tag {
	case "!!map", "!!seq", "!!str", "!!int", "!!float", "!!bool", "!!null":
	default:
		return fmt.Errorf("unsupported YAML tag %q", n.Tag)
	}
	for _, child := range n.Content {
		if err := plainYAML(child); err != nil {
			return err
		}
	}
	return nil
}
