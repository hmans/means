# Working on means

means is a general-purpose Go tool for managing collections of Markdown documents with
typed frontmatter and a collection-local schema. Read [README.md](README.md) for
the project direction. Track planned work in the [issues](issues/) collection,
following its [workflow](issues/MEANS.md).

## Core decisions

- Ship a single Go binary. Ordinary commands run to completion. The optional
  stdio MCP command lives for the client's connection; no managed daemon or
  network listener is required.
- Markdown files and their schemas are the source of truth. Humans and other
  tools can edit them directly.
- A collection has a schema describing its documents' application data. Consumers
  can establish and evolve that schema through means.
- Each collection contains one `MEANS.md`: YAML frontmatter with optional string
  title/description and required schema (JSON Schema object or boolean expressed
  as YAML, Draft 2020-12 by default), followed by Markdown guidance. Unknown
  configuration fields are rejected. Embedding settings belong in the user config, not MEANS.md (see docs/guide.md). Missing/blank title defaults to the directory
  name. Schema annotations do not define collection identity.
  `init` generates an example, accepts a complete file through `--input FILE|-`,
  or accepts JSON Schema through `--schema FILE|-`; these flags are exclusive.
  Validate before creating files; never overwrite MEANS.md. Init JSON returns
  its path in config. Complete file input preserves exact bytes. Schema input becomes YAML.
  Consumers use schema set or edit schemas directly. Schema set compiles the
  proposal and checks existing documents before replacing the schema. --force
  permits document diagnostics but never invalid schemas or filesystem failures.
  It does not migrate documents or prove compatibility for future data. Schemas
  can also be installed through MCP schema_set with the same validation and
  diagnostics; force is an optional boolean. A running server can repair an
  invalid or missing embedded schema if YAML and other fields are valid; malformed
  YAML requires direct editing. Startup requires valid configuration. Schemas are
  self-contained. Schema replacement preserves other field values, exact body bytes,
  and permissions; frontmatter may be reformatted. Observed changes to the whole
  configuration file reject replacement.
  Documents use YAML frontmatter and free-form Markdown bodies. The schema
  validates only frontmatter.data.
- Means owns the top-level frontmatter fields: id (generated, immutable), title
  (required nonempty string), and data (application object). Custom fields belong
  under data. Creation defaults omitted data to {}; explicit null is invalid.
- Creation generates a random 5-character lowercase ID and writes
  <id>-<title-slug>.md. Clients reference stored IDs, never filenames. Lookup scans
  visible subdirectories, ignores hidden paths and symlinks, and rejects duplicate
  IDs. Renames/moves preserve identity; title edits preserve the existing filename.
  Scans stop at subdirectories containing MEANS.md, even if that configuration is
  invalid. These are separate collections with independent IDs, schemas, and
  MEANS.md guidance. Adding/removing a nested MEANS.md changes subtree ownership.
  Creation reserves candidate IDs temporarily to prevent concurrent allocation;
  collisions retry. Direct edits can introduce duplicates, which list reports.
- Validate the means envelope and application data on creation and after updates.
  Reads and lists check the envelope but do not enforce the current data schema,
  allowing retrieval after schema changes. List reports malformed files and
  duplicate IDs separately in diagnostics, with paths, regardless of limits.
  List limits count valid documents only. Bodies are omitted by default; CLI
  list/search --json --with-body includes them for valid documents.
- Read/create/update JSON returns a SHA-256 hash of the exact stored file bytes.
  Updates require that hash, merge-patch title/data (null deletes nested keys, arrays replace),
  and replace the body only when supplied. Preserve IDs, reject stale writes, and
  use same-directory temporary replacements. Updates do not lock files; hash
  checks reject observed changes but cannot exclude a write racing replacement.
  Lists omit hashes so consumers retrieve full content before updating. Body-only
  changes preserve frontmatter text; metadata changes may reformat YAML.
- Support software consumers as well as humans and agents. Provide structured
  CLI input, output, and errors, and a stdio MCP interface; an SDK may follow.
- Use Cobra for the CLI. Every command inherits `--path` (defaulting to the
  current working directory) and `--json`. Use the shared output helpers for
  responses; write errors to stderr and successful results to stdout.
  List/search text output uses Charm Lip Gloss ID/title tables, with diagnostics
  below. Show renders title/ID and the Markdown body with Glamour; show --json
  equals read --json. Read text preserves exact source. Disable terminal styling
  for pipes and NO_COLOR; keep JSON and MCP responses independent of rendering.
- MCP uses the official Go SDK and calls the collection package directly. Bind
  each server to one collection, reload its schema per call, and preserve JSON
  number precision. Successful MCP results always use structuredContent without
  text copies; errors retain explanations. Keep stdout exclusively for MCP messages; --json affects
  help and startup errors, not protocol framing. Use MEANS.md frontmatter title/description for
  collection identity and its Markdown body for collection workflows.
  Root MEANS.md is reserved configuration, excluded from document lists/reads.
  MCP advertises identity and workflow at startup; info reads current
  guidance. The server does not load AGENTS.md. Multiple registrations may bind
  one agent to multiple collections, with names chosen by the consumer.
- Keep domain-specific concepts and workflows in schemas and skills. Issue
  tracking and bot memory are applications built on means.
- Full-text search uses a disposable Bleve index in each collection's
  .means/search/. Each search locks, scans/hashes visible documents, reconciles
  entries, queries, and closes the index. No watcher or daemon. Cache format
  changes trigger rebuilds. Literal and data-only searches use direct scans.
  Full-text scans reuse parsed frontmatter and body offsets from
  .means/documents.jsonl after checking full-file SHA-256 hashes. Its versioned,
  checksummed records are replaced atomically under search.lock; missing or
  corrupt metadata rebuilds. Invalid documents are reparsed. Bump the metadata
  cache version when parsing rules change. Other operations use direct scans.
  Datetime search filters compare RFC 3339 string fields directly under data by
  instant, with before/after and inclusive on_or_before/on_or_after operators.
  Combine them with exact filters using AND before limits in all search modes.
  Require timezone offsets and at most nine fractional digits; reject invalid
  filter inputs and skip missing, non-string, or invalid document values.
  CLI exposes repeatable --before/--after/--on-or-before/--on-or-after KEY=TIMESTAMP;
  MCP exposes datetime arrays of {field, op, value}. Exact equality is unchanged.
  Full-text queries AND lowercase Unicode tokens across title/body, boost title
  matches, and sort score ties by ID. No stemming or stop-word removal. Data
  filters keep exact JSON semantics and apply before limits. Init ensures a
  .gitignore rule for .means/ while preserving existing content.

## Development

- Use the Go toolchain configured in `mise.toml`.
- Keep changes small and focused on the current milestone. Avoid building
  speculative abstractions or infrastructure for future features.
- Record user-visible changes with `mise sweet change .` and commit the generated
  `.sweet/` note with the code. See [docs/releases.md](docs/releases.md) for release
  policies and the tagging and publishing workflow.
- Keep document operations independent of CLI presentation so other interfaces
  can reuse the same behavior.
- Once Go code exists, format it with `gofmt` and run relevant tests using
  `mise test`.
- Use means to manage this repository's issues. Prefer the `issues` MCP server
  configured in root `.codex/config.toml`; call `info` to confirm its identity is
  "means issues", then list/read relevant issues and update status and outcomes.
  CLI fallback from the repo root: `mise means list --path ./issues --limit 0 --json`.
- Keep root README.md concise: explain what means is, its current capabilities,
  and where to start. Put detailed workflows, command contracts, and migration
  guidance in docs/ or CLI help instead of expanding the README. Update the
  relevant documentation when behavior changes, and distinguish available
  features from plans. TODO.md retains historical milestones only.
- Keep CLI help sufficient for agents to discover workflows, input formats,
  examples, output formats, errors, and limitations. Ordinary command help is
  concise; `help agents <command>` and JSON help include full contracts and examples.
  Bare invocation, help, and version never load or bootstrap user configuration.
  `info` exposes collection identity, schema, and exact workflow body. `show --raw`
  exposes exact document source; the hidden `read` command keeps its old behavior.
  List accepts the same metadata filters as search. `create --title` with optional
  `--body-file` uses empty data and is exclusive with structured `--input`.
  `delete --hash` is an alternative to its structured input, with the same checks.
  `edit` opens a private temporary copy using VISUAL/EDITOR (executable only),
  reloads the schema, and saves through the shared hash-checked replacement path.
  Edited metadata replaces rather than merge-patches; preserve IDs and literal nulls.
  Failed edits retain the temporary file for recovery.
- Delete requires the document ID and current content hash. It permanently removes
  only the document file, preserving directories and configuration. Missing and
  duplicate IDs, malformed envelopes, and stale hashes fail. Existing application
  data need not satisfy the current schema. As with updates, concurrent writes
  can race removal; no force option or automatic retries. Opted-in embedding
  operations share a cache lock, but external document edits do not take it.
- Embeddings are opt-in through the machine-local user config embeddings field
  (object, or false to disable), applying to every collection. Config lives at
  $XDG_CONFIG_HOME/means/config.yaml, default ~/.config/means/config.yaml.
  Collection commands/MCP startup bootstrap a commented, disabled example if missing;
  never overwrite existing files. Bootstrap failures warn and allow offline use;
  existing invalid/unreadable configs fail (CLI help/version remain available).
  CLI reads settings per invocation; MCP captures them at startup. Collections
  receive host settings explicitly; the collection package never loads user files.
  Reject legacy MEANS.md embeddings with migration instructions. No config command,
  automatic migration, per-collection overrides, or automatic search backfill.
  Require provider openai or openrouter; default endpoint/model/key variable follows
  that provider. OpenAI text-embedding-3-small/large only (openai/ prefix required
  for OpenRouter). Environment overrides may change endpoint/key; keys never fall
  back across providers. Provider/model/dimension/endpoint changes invalidate vectors.
  Default to the provider's OpenAI text-embedding-3-small model and 512 dimensions.
  Defaults apply in memory; user config and MEANS.md are not rewritten.
  CLI JSON mode suppresses embedding progress to keep errors valid JSON.
  Refresh changed documents in sequential batches of at most 32; validate all response
  indices/vectors, then checkpoint once per batch. Failed batches preserve earlier work.
  Embed is CLI-only; MCP exposes no embed tool. Embed --force clears stored vectors under the cache lock
  before regenerating all valid documents. Failures retain only batches completed
  in that run; ordinary embed resumes without discarding those batches.
  Embed title plus exact body into one float vector. Store in a separate versioned,
  checksummed .means/embeddings.jsonl, serialized under embeddings.lock. Create/update
  persist the vector before the document; API/cache failures prevent document writes.
  Delete removes its vector before the file, without an API call. Failed source
  writes/crashes can leave stale derived data; embed reconciles against visible
  documents, including direct edits and nested boundaries. Search embeds only the
  query and never changes the embedding cache; it skips missing/stale vectors and
  invisible documents. It reads the atomically replaced cache without taking the lock.
  Data-only edits reuse vectors. Semantic queries use cosine ranking,
  exact data filters before limits, and ID ties. The inclusive minimum cosine score
  defaults to 0.3; --min-score / MCP min_score override it from -1 to 1 and require
  semantic search. Apply the threshold before limits/has_more. Semantic text tables
  display scores to three decimals; filtering uses full precision.
  No chunking, quantization, sharing,
  or hybrid search yet. Read/list/full-text/literal operations remain offline.
- Automatic document repair is deferred.
- Doctor checks opted-in embedding caches offline without locking or writing.
  Missing/stale document vectors, orphaned vectors, and corrupt/incompatible caches
  are warnings that do not change healthy or the exit code; JSON warnings are
  separate from document diagnostics. Cache read failures still fail the check.
  Schema-invalid data still owns
  its vector when the envelope and ID are valid and unique. Checked counts files.
