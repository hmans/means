# Working on means

means is a Go tool for agentic memory: collections of Markdown documents with
typed frontmatter and a collection-local schema. Read [README.md](README.md) for
the project direction and [TODO.md](TODO.md) for planned work.

## Core decisions

- Ship a single Go binary. Ordinary commands run to completion. The optional
  stdio MCP command lives for the client's connection; no managed daemon or
  network listener is required.
- Markdown files and their schemas are the source of truth. Humans and other
  tools can edit them directly.
- A collection has a schema describing its documents' frontmatter. Consumers
  can establish and evolve that schema through means.
- Each collection contains a standalone `schema.json` using JSON Schema (Draft
  2020-12 by default). `init` creates the directory with an example schema or a
  consumer-authored schema from `--schema FILE` / `--schema -` (stdin). Validate
  supplied schemas before creating files; never overwrite an existing schema.
  Consumers edit schemas directly. Schemas are currently self-contained.
  Documents use YAML frontmatter and free-form Markdown bodies.
- Document IDs are filename stems. Creation generates a random 32-character hex
  ID and writes `<id>.md`; IDs are not injected into application frontmatter.
  Collections contain regular Markdown files directly under the selected path.
- Validate frontmatter on creation and after updates. Reads and lists parse stored metadata without
  enforcing the schema, allowing retrieval after schema changes. List reports
  malformed files individually and omits document bodies.
- Read/create/update JSON returns a SHA-256 hash of the exact stored file bytes.
  Updates require that hash, merge-patch frontmatter (null deletes, arrays replace),
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
- MCP uses the official Go SDK and calls the collection package directly. Bind
  each server to one collection, reload its schema per call, and preserve JSON
  number precision. Keep stdout exclusively for MCP messages; --json affects
  help and startup errors, not protocol framing. Use schema title/description for
  collection identity and optional instructions.md for collection workflows.
  instructions.md is reserved configuration, excluded from document lists/reads.
  MCP advertises identity and workflow at startup; info reads current
  guidance. The server does not load AGENTS.md. Multiple registrations may bind
  one agent to multiple collections, with names chosen by the consumer.
- Keep domain-specific concepts and workflows in schemas and skills. Issue
  tracking and bot memory are applications built on means.
- Internal storage is optional and derived. Its format is undecided.

## Development

- Use the Go toolchain configured in `mise.toml`.
- Keep changes small and focused on the current milestone. Avoid building
  speculative abstractions or infrastructure for future features.
- Keep document operations independent of CLI presentation so other interfaces
  can reuse the same behavior.
- Once Go code exists, format it with `gofmt` and run relevant tests using
  `mise test`.
- Update the README when implemented behavior changes and check off completed
  work in TODO.md. Clearly distinguish available features from plans.
- Keep CLI help sufficient for agents to discover workflows, input formats,
  examples, output formats, errors, and limitations. Include long descriptions
  and examples in both human-readable and JSON help.
- Delete semantics are still an open design question.
- Embeddings, vector search, and automatic document repair are deferred.
