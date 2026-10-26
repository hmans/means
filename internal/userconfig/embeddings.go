package userconfig

import (
	"encoding/json"
	"fmt"
)

// Embeddings opts this host into network-backed embedding maintenance.
// Credentials and endpoint overrides belong in the process environment.
type Embeddings struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
}

func parseEmbeddings(value any) (*Embeddings, error) {
	if value == false {
		return nil, nil
	}
	fields, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("embeddings must be an object or false")
	}
	cfg := &Embeddings{}
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
	prefix := ""
	switch cfg.Provider {
	case "openai":
	case "openrouter":
		prefix = "openai/"
	default:
		return nil, fmt.Errorf("embeddings.provider is required and must be openai or openrouter")
	}
	if cfg.Model == "" {
		cfg.Model = prefix + "text-embedding-3-small"
	}
	maxDimensions := 1536
	switch cfg.Model {
	case prefix + "text-embedding-3-small":
	case prefix + "text-embedding-3-large":
		maxDimensions = 3072
	default:
		return nil, fmt.Errorf("embeddings.model for %s must be %s or %s", cfg.Provider, prefix+"text-embedding-3-small", prefix+"text-embedding-3-large")
	}
	if cfg.Dimensions == 0 {
		cfg.Dimensions = 512
	}
	if cfg.Dimensions > maxDimensions {
		return nil, fmt.Errorf("embeddings.dimensions exceeds model maximum %d", maxDimensions)
	}
	return cfg, nil
}
