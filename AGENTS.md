# Working on means

means is a Go tool for agentic memory: collections of Markdown documents with
typed frontmatter and a collection-local schema. Read [README.md](README.md) for
the project direction and [TODO.md](TODO.md) for planned work.

## Core decisions

- Ship a single Go binary. Commands run to completion; no daemon or other
  long-running process is required.
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
- Validate frontmatter on creation. Reads and lists parse stored metadata without
  enforcing the schema, allowing retrieval after schema changes. List reports
  malformed files individually and omits document bodies.
- Support software consumers as well as humans and agents. Plan for structured
  CLI input, output, and errors; SDK and MCP interfaces may follow.
- Use Cobra for the CLI. Every command inherits `--path` (defaulting to the
  current working directory) and `--json`. Use the shared output helpers for
  responses; write errors to stderr and successful results to stdout.
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
- Update and delete semantics are still open design questions. Do not treat
  examples from discussion as settled APIs.
- Embeddings, vector search, and automatic document repair are deferred.
