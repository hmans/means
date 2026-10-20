# means

An agentic memory substrate built from Markdown files.

means will provide a CLI for creating, reading, updating, searching, and deleting
documents in a directory. Each document has a Markdown body and typed
frontmatter, governed by a schema stored alongside the collection. Applications
and agents can establish and evolve that schema as their needs change.

**Status:** early development. Collection initialization, document creation,
reading, and metadata listing are implemented. Updates, deletion, search, and
`doctor` are planned.

## What it's for

means provides storage, validation, and retrieval primitives. Schemas and skills
give those primitives domain-specific meaning and workflows. For example:

- An issue-tracking skill could define fields such as status and priority and
  teach an agent how to manage work, similar to [beans](https://github.com/hmans/beans).
- A bot such as ChattoBot could use means to retain knowledge across conversations.

Software integration is a priority. The CLI supports JSON responses for
applications in any language, with SDK and MCP interfaces under consideration.

## CLI

Every command accepts these shared flags before or after the command name:

- `--path <directory>` selects the directory to work with, defaulting to the
  current working directory. Relative paths are resolved against that directory;
  means does not change the process working directory.
- `--json` returns JSON with two-space indentation, including help and error responses.

Successful responses go to stdout. Errors go to stderr and cause a nonzero exit
code; in JSON mode, errors have an `error` string. Directory existence is left to
the commands that access it; help and version do not require a collection.

```sh
means --help
means version
means version --path ./notes --json
```

The last command currently returns:

```json
{
  "version": "dev"
}
```

## Collections and schemas

Initialize a collection with an example frontmatter schema:

```sh
mise means init --path ./memory
```

By default, `init` creates the directory if needed and writes an example `schema.json`. It
refuses to overwrite an existing schema and leaves other files alone. Without
`--path`, it initializes the current directory. Collection loading uses exactly
the selected directory; it does not search parent directories.

Edit `schema.json` directly to define the collection's frontmatter. The file is
a standalone JSON Schema:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "title": { "type": "string" },
    "tags": { "type": "array", "items": { "type": "string" } }
  },
  "required": ["title"],
  "additionalProperties": false
}
```

The starter schema requires a string `title`, accepts an optional list of string
`tags`, and rejects undeclared fields. These are starter conventions; collections
can define their own fields. Documents use YAML frontmatter and free-form
Markdown bodies.

Agents can author the schema first and pass it to `init` in one operation:

```sh
# Read a standalone JSON Schema from a file
mise means init --path ./memory --schema ./memory.schema.json --json

# Or read it from stdin until EOF
mise means init --path ./memory --schema - --json <<'JSON'
{
  "type": "object",
  "properties": {
    "summary": { "type": "string", "description": "A fact worth remembering." }
  },
  "required": ["summary"],
  "additionalProperties": false
}
JSON
```

The input is the schema itself, without a configuration wrapper. Relative schema
file paths resolve against the current working directory, independently of
`--path`. Supplied schemas are validated before any directory or file is created
and are written unchanged. Empty or invalid input fails without falling back to
the example. An existing `schema.json` is never overwritten.

Successful initialization returns absolute `path` and `schema` paths on stdout
in JSON mode. Failures return an `error` string on stderr with exit code 1. Agents
can correct invalid schemas and retry. To evolve an existing schema, edit
`schema.json` directly.

`mise means init --help` includes the complete workflow, schema conventions,
examples, and response formats. The same instructions are available through
`mise means help init --json` in the `description` and `usage` fields.

Schemas default to JSON Schema Draft 2020-12 when `$schema` is omitted. Internal
references such as `#/$defs/status` work; references to other files or remote
schemas are not supported yet.

The collection loader reads and validates `schema.json` on each load, so manual
schema edits take effect immediately. This checks the schema definition; checking
existing Markdown documents will be the job of the planned `doctor` command.

## Documents

Create a document from a JSON envelope containing application metadata and a
Markdown body. `--input` accepts a file path or `-` for stdin:

```sh
mise means create --path ./memory --input - --json <<'JSON'
{
  "frontmatter": {
    "title": "Use Go for means",
    "tags": ["decision"]
  },
  "body": "We chose Go for single-binary distribution.\n"
}
JSON
```

This example uses the default schema. Adapt the frontmatter to your collection's
`schema.json`. The `frontmatter` object is required; `body` is an optional string
that defaults to empty. Additional envelope fields are rejected. Input file paths
resolve against the current working directory, independently of `--path`.

Creation validates metadata before writing, generates a random 32-character
hexadecimal ID, and stores `<id>.md`. The ID is only the filename stem; means does
not impose an `id` or `title` field on application metadata. Renaming the file
changes the document's ID. A successful create is not idempotent: repeating it
creates another document.

The stored file is ordinary Markdown:

```markdown
---
tags:
  - decision
title: Use Go for means
---
We chose Go for single-binary distribution.
```

With `--json`, creation and reading return `{ "id": "...", "frontmatter": {...},
"body": "..." }`. Text-mode creation prints just the ID; text-mode reading prints
the exact stored Markdown. Body content and trailing newlines are preserved.

```sh
mise means read <id> --path ./memory --json
mise means list --path ./memory --json
mise means list --path ./memory --limit 0 --json
```

Listing returns `{ "documents": [{ "id": "...", "frontmatter": {...} }],
"has_more": false }`, without bodies. Results are ordered by filename. The default
limit is 20; `--limit 0` returns all documents. Increase the limit when `has_more`
is true. Text output contains one ID and its metadata per line.

Manually created `.md` files work too. Filenames must have a stem beginning with
a letter or digit, followed by letters, digits, hyphens, or underscores. Listing
only includes regular files directly in the collection; it skips symlinks,
subdirectories, and other filenames. Reads reject paths and symlinks.

Reads and lists parse metadata without enforcing the current schema, so documents
remain retrievable after schema changes. A valid collection schema is still
required. Malformed documents appear in list results with an `error` and null
`frontmatter` rather than blocking the entire list; those entries count toward
the limit and do not change the successful list exit code. Direct reads of
malformed documents fail.

Frontmatter uses string keys and JSON-compatible values. Duplicate keys, aliases,
merge keys, custom YAML tags, and non-finite numbers are rejected. Plain YAML dates
are returned as strings. Both LF and CRLF delimiters are supported.

Run `mise means create --help`, `mise means read --help`, or `mise means list
--help` for full input, output, and error details. Help is also available as JSON.

## Design direction

- **Plain files:** Markdown documents and collection schemas remain authoritative,
  editable, and suitable for version control.
- **Typed frontmatter:** schemas describe document metadata and can evolve through
  explicit consumer operations.
- **Single binary:** implemented in Go, with short-lived commands and no required
  resident process.
- **Optional indexes:** means may persist derived data locally to support efficient
  retrieval. The storage format is undecided.
- **Diagnostics:** a planned `doctor` command will identify documents that do not
  satisfy their collection's schema. A possible `fix` command is future work.

See [TODO.md](TODO.md) for the next steps, including updates and deletion.

## Development

The Go toolchain is managed through [mise](https://mise.jdx.dev/) and configured in
`mise.toml`. Run `mise install` to install the configured toolchain.

```sh
mise means help
mise means version --json
mise exec -- go test ./...
mise exec -- go build -o means .
```

`mise means` runs `go run .` and forwards all arguments to the CLI, including
`--help`, `--path`, and `--json`.

The entry point is `main.go`; command definitions and output formatting live in
`internal/cli`. Collection initialization, document storage, and schema validation live in
`internal/collection`, independently of the CLI. Development builds report `dev`
as their version. Release builds
can set it with `go build -ldflags "-X main.version=<version>"`.

See [AGENTS.md](AGENTS.md) for repository guidance.
