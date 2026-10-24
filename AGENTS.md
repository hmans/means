# Working on means

means is a Go tool for agentic memory: collections of Markdown documents with
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
- Each collection contains a standalone `means.schema.json` using JSON Schema (Draft
  2020-12 by default). `init` creates the directory with an example schema or a
  consumer-authored schema from `--schema FILE` / `--schema -` (stdin). Validate
  supplied schemas before creating files; never overwrite an existing schema.
  Consumers use schema set or edit schemas directly. Schema set compiles the
  proposal and checks existing documents before replacing the schema. --force
  permits document diagnostics but never invalid schemas or filesystem failures.
  It does not migrate documents or prove compatibility for future data. Schemas
  can also be installed through MCP schema_set with the same validation and
  diagnostics; force is an optional boolean. A running server can repair an
  externally invalidated schema, while startup requires a valid schema. Schemas
  are currently self-contained.
  Documents use YAML frontmatter and free-form Markdown bodies. The schema
  validates only frontmatter.data; schema title/description still describe the collection.
- Means owns the top-level frontmatter fields: id (generated, immutable), title
  (required nonempty string), and data (application object). Custom fields belong
  under data. Creation defaults omitted data to {}; explicit null is invalid.
- Creation generates a random 5-character lowercase ID and writes
  <id>-<title-slug>.md. Clients reference stored IDs, never filenames. Lookup scans
  visible subdirectories, ignores hidden paths and symlinks, and rejects duplicate
  IDs. Renames/moves preserve identity; title edits preserve the existing filename.
  Scans stop at subdirectories containing means.schema.json, even if that schema is
  invalid. These are separate collections with independent IDs, schemas, and
  MEANS.md guidance. Adding/removing a nested schema changes subtree ownership.
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
  help and startup errors, not protocol framing. Use schema title/description for
  collection identity and optional MEANS.md for collection workflows.
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
  examples, output formats, errors, and limitations. Include long descriptions
  and examples in both human-readable and JSON help.
- Delete requires the document ID and current content hash. It permanently removes
  only the document file, preserving directories and configuration. Missing and
  duplicate IDs, malformed envelopes, and stale hashes fail. Existing application
  data need not satisfy the current schema. As with updates, concurrent writes
  can race removal; there are no locks, force option, or automatic retries.
- Embeddings, vector search, and automatic document repair are deferred.
