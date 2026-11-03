package collection

import "fmt"

// EmbeddingConfig opts the host into network-backed embedding maintenance.
// Credentials and endpoint overrides come from the process environment.
type EmbeddingConfig struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
}

// WithDefaults returns validated settings without changing the receiver.
// Provider is required. Empty Model selects text-embedding-3-small (with the
// openai/ prefix for OpenRouter); zero Dimensions selects 512.
func (cfg EmbeddingConfig) WithDefaults() (EmbeddingConfig, error) {
	prefix := ""
	switch cfg.Provider {
	case "openai":
	case "openrouter":
		prefix = "openai/"
	default:
		return EmbeddingConfig{}, fmt.Errorf("embeddings.provider is required and must be openai or openrouter")
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
		return EmbeddingConfig{}, fmt.Errorf("embeddings.model for %s must be %s or %s", cfg.Provider, prefix+"text-embedding-3-small", prefix+"text-embedding-3-large")
	}
	if cfg.Dimensions == 0 {
		cfg.Dimensions = 512
	}
	if cfg.Dimensions < 1 || cfg.Dimensions > maxDimensions {
		return EmbeddingConfig{}, fmt.Errorf("embeddings.dimensions must be an integer from 1 to %d", maxDimensions)
	}
	return cfg, nil
}
