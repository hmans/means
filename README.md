# means

An agentic memory substrate built from Markdown files.

means will provide a CLI for creating, reading, updating, searching, and deleting
documents in a directory. Each document has a Markdown body and typed
frontmatter, governed by a schema stored alongside the collection. Applications
and agents can establish and evolve that schema as their needs change.

**Status:** early development. Collection initialization, document creation,
reading, metadata listing, and a stdio MCP server are implemented. Updates, deletion, search, and
`doctor` are planned.

## What it's for

means provides storage, validation, and retrieval primitives. Schemas and skills
give those primitives domain-specific meaning and workflows. For example:

- An issue-tracking skill could define fields such as status and priority and
  teach an agent how to manage work, similar to [beans](https://github.com/hmans/beans).
- A bot such as ChattoBot could use means to retain knowledge across conversations.

Software integration is a priority. The CLI supports JSON responses for
applications in any language, plus a stdio MCP interface. An SDK is under consideration.

## CLI

Every command accepts these shared flags before or after the command name:

- `--path <directory>` selects the directory to work with, defaulting to the
  current working directory. Relative paths are resolved against that directory;
  means does not change the process working directory.
- `--json` returns JSON with two-space indentation, including help and error responses.

The `mcp` command uses MCP message framing on stdout regardless of `--json`;
the flag still controls its help and startup error formatting.

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
subdirectories, and other filenames. Reads reject paths and symlinks. The reserved
`instructions.md` contains collection guidance and is excluded from document
lists and reads, even if it has frontmatter.

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

## MCP

Launch a server bound to one existing collection:

```sh
means mcp --path /absolute/path/to/collection
# From this checkout:
mise means mcp --path ./example/issues
```

An MCP client launches this command as a subprocess and communicates over
stdin/stdout. The process exits when stdin closes. No network listener or
separately managed daemon is needed. The official
[Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) handles protocol
compatibility, including the 2026-07-28 revision and older clients.

| Tool | Arguments | Result |
| --- | --- | --- |
| `info` | `{}` | Collection title, description, path, current schema, usage instructions |
| `list` | Optional `limit` (20 by default, 0 for all) | `documents` containing IDs and metadata, plus `has_more` |
| `read` | `id` | ID, frontmatter, body |
| `create` | Required `frontmatter`, optional `body` | Created ID, frontmatter, body |

All tools use the configured collection; they do not accept arbitrary paths.
Call `info` before creating documents to learn the frontmatter schema.
The server reloads the schema on every tool call, so edits take effect immediately.
Validation, IDs, listing limits, malformed-file handling, and number preservation
match the CLI. Successful creation is not idempotent.

Results contain `structuredContent` and an equivalent pretty-printed JSON text
block. Tool failures return MCP `isError` results without ending the server.
Stdout contains only compact, newline-delimited MCP messages. Diagnostics and
startup failures go to stderr. `--json` never changes protocol framing.

Initialize collections and edit schemas/instructions through the CLI/filesystem;
these are not MCP tools yet. The server does not load `AGENTS.md` itself.
Run `means mcp --help` for the full contract.

### Collection identity and instructions

Give the schema a top-level `title` and `description` to tell agents what belongs
in this collection. These annotations describe the collection, separately from
document fields inside `properties`:

```json
{
  "title": "Project issues",
  "description": "Bugs, features, and planned work for this project.",
  "type": "object",
  "properties": { "title": { "type": "string" } },
  "required": ["title"]
}
```

The title and purpose appear in the MCP server metadata, server instructions, and
every tool description. Agents can choose the right collection before calling a
tool. An omitted or blank title falls back to the directory name; description
is optional. Boolean schemas also work with these fallbacks.

Optionally add a plain UTF-8 `instructions.md` beside `schema.json`:

```markdown
Check existing issues for duplicates before creating one.
Include concrete acceptance criteria in each issue's body.
```

This file needs no frontmatter. It supplies the workflow in MCP server instructions
and the `info` result's `instructions` field, alongside general means
usage guidance. It must be a regular file, not a directory or symlink. Missing or
empty instructions are allowed; unreadable or invalid instructions fail MCP
startup or `info` with an explanation. `init` still creates only
`schema.json`. Other document operations do not depend on this optional file.

Discovery metadata and connection instructions reflect startup state. Restart
the server/client after editing title, description, or instructions to refresh
those advertised values. `info` always reads current guidance;
schema validation and document reads also use current files without a restart.

### Multiple brains for one agent

Register `means mcp` once per collection, using names you choose in the client.
For example, with an installed `means` binary on PATH, a Codex configuration could
contain the following (replace paths with your actual collection directories):

```toml
[mcp_servers.issues]
command = "means"
args = ["mcp", "--path", "/absolute/project/issues"]

[mcp_servers.decisions]
command = "means"
args = ["mcp", "--path", "/absolute/project/decisions"]
```

Each registration exposes the same four tool names, bound to its own collection.
The client registration distinguishes the servers; schema annotations distinguish
their purpose, and each collection supplies its own workflow. No routing flag or
additional means configuration format is needed.

## Example issue tracker

An issue-tracking example lives in [example/issues](example/issues), with a schema
and fictional sample issues. Its [instructions.md](example/issues/instructions.md)
supplies an issue workflow through MCP, without an example-local `AGENTS.md`.
For CLI access:

```sh
mise means list --path ./example/issues --limit 0 --json
```

The example includes a [project-local Codex MCP configuration](example/.codex/config.toml).
See [example/README.md](example/README.md) to start Codex with the issue collection
connected and its agent instructions loaded.

## Design direction

- **Plain files:** Markdown documents and collection schemas remain authoritative,
  editable, and suitable for version control.
- **Typed frontmatter:** schemas describe document metadata and can evolve through
  explicit consumer operations.
- **Single binary:** implemented in Go, with short-lived CLI commands and an
  optional MCP subprocess managed by its client.
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
mise test
mise exec -- go build -o means .
```

`mise means` runs the repository's Go package from your current working directory
and forwards all arguments to the CLI, including `--help`, `--path`, and `--json`.
You can work directly inside a collection without specifying `--path`:

```sh
cd example/issues
mise means list --json
```

The entry point is `main.go`; command definitions and output formatting live in
`internal/cli`. Collection initialization, document storage, and schema validation live in
`internal/collection`, independently of the CLI. Development builds report `dev`
as their version. Release builds
can set it with `go build -ldflags "-X main.version=<version>"`.

See [AGENTS.md](AGENTS.md) for repository guidance.
