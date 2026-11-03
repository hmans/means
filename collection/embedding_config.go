package collection

import "fmt"

// EmbeddingConfig opts the host into network-backed embedding maintenance.
// Attach it through Collection.WithEmbeddings. Creation and title/body updates
// can call the provider and fail before writing the document; data-only edits
// reuse unchanged valid vectors. Collection.Embed also backfills directly edited
// files.
//
// Credentials come from OPENAI_API_KEY for provider "openai" and
// OPENROUTER_API_KEY for "openrouter". A nonempty MEANS_EMBEDDING_API_KEY
// overrides either. Keys never fall back across providers. Requests use
// https://api.openai.com/v1 or https://openrouter.ai/api/v1 by provider; a nonempty
// MEANS_EMBEDDING_BASE_URL overrides the base URL (before /embeddings).
// The override must be HTTP(S) without user credentials, a query, or a fragment.
// The host must load any .env files itself; network operations read the process
// environment when they run. These settings do not change MEANS.md or user config.
type EmbeddingConfig struct {
	// Provider is required: "openai" or "openrouter".
	Provider string `json:"provider"`
	// Model is text-embedding-3-small or text-embedding-3-large, with the openai/
	// prefix required for OpenRouter. Empty selects the provider's small model.
	Model string `json:"model"`
	// Dimensions is the vector size: 1–1536 for the small model, or 1–3072 for
	// the large model. Zero selects 512.
	Dimensions int `json:"dimensions"`
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
