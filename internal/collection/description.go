package collection

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// Description gives consumers the collection identity and Markdown guidance.
type Description struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
}

// configFields validates configuration independently of the application schema.
// Schema replacement uses this to repair an invalid or missing schema.
func configFields(data []byte) (map[string]any, string, error) {
	fields, body, err := parseMarkdown(data)
	if err != nil {
		return nil, "", err
	}
	for key, value := range fields {
		switch key {
		case "schema":
		case "embeddings":
			return nil, "", fmt.Errorf("embeddings settings have moved to the Means user config ($XDG_CONFIG_HOME/means/config.yaml, default ~/.config/means/config.yaml); move this block there and remove it from MEANS.md")
		case "title", "description":
			if _, ok := value.(string); !ok {
				return nil, "", fmt.Errorf("%s must be a string", key)
			}
		default:
			return nil, "", fmt.Errorf("unknown collection configuration field %q", key)
		}
	}
	return fields, body, nil
}

func parseConfig(path string, data []byte) (*Collection, error) {
	fields, body, err := configFields(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ConfigFilename, err)
	}
	switch fields["schema"].(type) {
	case map[string]any, bool:
	default:
		return nil, fmt.Errorf("%s: schema is required and must be an object or boolean", ConfigFilename)
	}
	schema, err := json.Marshal(fields["schema"])
	if err != nil {
		return nil, err
	}
	c, err := compile(path, schema)
	if err != nil {
		return nil, err
	}
	c.config, c.body = fields, body
	return c, nil
}

// encodeConfig writes identity fields before the schema so MEANS.md reads top-down.
func encodeConfig(fields map[string]any, body string) (string, error) {
	return encodeMarkdown(fields, body, "title", "description", "schema")
}

// Describe returns identity and guidance from the loaded configuration snapshot.
func (c *Collection) Describe() *Description {
	info := &Description{Title: filepath.Base(c.Path), Instructions: c.body}
	if title, ok := c.config["title"].(string); ok && strings.TrimSpace(title) != "" {
		info.Title = strings.TrimSpace(title)
	}
	if description, ok := c.config["description"].(string); ok {
		info.Description = strings.TrimSpace(description)
	}
	return info
}
