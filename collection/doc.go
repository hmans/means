// Package collection manages Markdown documents with stable IDs, typed
// application data, and a collection-local JSON Schema in MEANS.md.
//
// Init and InitConfig create collections without overwriting MEANS.md. Load
// reads the exact directory supplied and compiles a configuration snapshot.
// Documents are scanned on each operation; configuration is not automatically
// reloaded. Call Load again after schema, identity, views, or instructions change.
// Collection values must be obtained through these constructors.
//
// Operations take a non-nil context. Cancellation is checked before work starts
// and during embedding requests and cache lock waits. Synchronous filesystem
// work and schema validation may finish before cancellation is observed.
// A loaded Collection can be shared by concurrent callers. WithEmbeddings and
// WithEmbeddingProgress return copies; progress callbacks can run concurrently
// and must synchronize their own state.
//
// Create, Update, and ReplaceMarkdown validate application data. Read, List, and
// Search check document envelopes without enforcing the loaded data schema.
// Use Doctor to check all documents against that schema. List, Search, and Doctor
// return document diagnostics separately from fatal errors.
//
// Update and Delete require the hash from Read. Stale hashes return ErrConflict;
// missing IDs wrap fs.ErrNotExist and duplicate IDs wrap ErrDuplicateID. Hash
// checks reject observed changes but cannot exclude external writes racing the
// final replacement or deletion. Creation is not idempotent.
//
// The package does not load user configuration or .env files and does not print
// output. Embeddings are disabled until the host supplies an EmbeddingConfig.
// Enabled embedding operations read credentials and endpoint overrides from the
// process environment. Full-text search writes disposable caches under .means/;
// ordinary reads, lists, and literal searches remain offline.
package collection
