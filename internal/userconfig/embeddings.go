package userconfig

import (
	"encoding/json"
	"fmt"

	"github.com/hmans/means/collection"
)

func parseEmbeddings(value any) (*collection.EmbeddingConfig, error) {
	if value == false {
		return nil, nil
	}
	fields, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("embeddings must be an object or false")
	}
	cfg := &collection.EmbeddingConfig{}
	for key, value := range fields {
		switch key {
		case "provider":
			provider, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("embeddings.provider must be openai or openrouter")
			}
			cfg.Provider = provider
		case "model":
			model, ok := value.(string)
			if !ok || model == "" {
				return nil, fmt.Errorf("embeddings.model must be a string")
			}
			cfg.Model = model
		case "dimensions":
			n, ok := value.(json.Number)
			if !ok {
				return nil, fmt.Errorf("embeddings.dimensions must be a positive integer")
			}
			i, err := n.Int64()
			if err != nil || i < 1 || i > 3072 {
				return nil, fmt.Errorf("embeddings.dimensions must be an integer from 1 to 3072")
			}
			cfg.Dimensions = int(i)
		default:
			return nil, fmt.Errorf("unknown embeddings configuration field %q", key)
		}
	}
	resolved, err := cfg.WithDefaults()
	if err != nil {
		return nil, err
	}
	return &resolved, nil
}
