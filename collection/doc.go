// Package collection manages Markdown documents with stable IDs, typed
// application data, and a collection-local JSON Schema in MEANS.md.
//
// # Getting started
//
// Init and InitConfig create collections without overwriting MEANS.md. Load
// reads the exact directory supplied and compiles a configuration snapshot.
// Documents are scanned on each operation; configuration is not automatically
// reloaded. Call Load again after schema, identity, views, or instructions change.
// Collection values must be obtained through these constructors; the zero value
// is not usable. No Close call is needed: each operation closes its files and
// caches before returning. See the examples for [Collection.Create],
// [Collection.Search], [Collection.Update], and [SetSchema].
// [Watch] provides optional change notifications with a separate context-owned
// lifetime. Cancel its context to release filesystem watches.
//
// MEANS.md contains YAML frontmatter with a required schema and optional title,
// description, and app views, followed by Markdown workflow instructions.
// Documents contain only id, title, and data in their YAML frontmatter, followed
// by a free-form body. The schema validates data, not the means-owned fields.
// Visible Markdown files in subdirectories belong to the collection, except
// below another MEANS.md. Hidden paths and symlinks are ignored. MEANS.md itself
// is configuration, not a document.
//
// # Inputs and results
//
// Creation, updates, validation, and list/search filters accept JSON-compatible
// Go values such as int and []string without changing input maps. Stored values
// use map[string]any, []any, and json.Number. Supply json.Number to preserve exact
// large integers and decimals. Do not mutate inputs while an operation uses them.
// List and search summaries omit hashes: read by ID before changing a document.
// Their optional body pointers distinguish an omitted body from an empty body.
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
// # Cancellation and host settings
//
// Operations take a non-nil context. Cancellation is checked before work starts
// and during embedding requests and cache lock waits. Synchronous filesystem
// work and schema validation may finish before cancellation is observed.
// A loaded Collection can be shared by concurrent callers. WithEmbeddings and
// WithEmbeddingProgress return copies; progress callbacks can run concurrently
// and must synchronize their own state.
//
// The package does not load user configuration or .env files and does not print
// output. Embeddings are disabled until the host supplies an EmbeddingConfig.
// Enabled embedding operations read credentials and endpoint overrides from the
// process environment as described by [EmbeddingConfig]. Full-text search writes
// disposable caches under .means/; ordinary reads, lists, and literal searches
// remain offline.
package collection
