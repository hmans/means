# means

An agentic memory substrate built from Markdown files.

means provides a CLI for creating, reading, updating, and searching
documents in a directory. Each document has a Markdown body and typed
frontmatter. Means owns document IDs and titles; a schema stored alongside the
collection governs application fields under `data`. Applications
and agents can establish and evolve that schema as their needs change.

**Status:** early development. Collection initialization, document creation,
reading, patch updates, metadata listing, text/data search, document diagnostics,
and a stdio MCP server are implemented. Deletion is planned.

This repository uses means to track its own work. The [issues](issues/) directory
is a means collection with its own [schema](issues/schema.json) and
[workflow](issues/instructions.md); issue status and outcomes are updated through
the same CLI and MCP interfaces available to other collections.

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

The `mcp` command uses MCP message framing on stdout regardless of `--json`.
The flag omits redundant text copies from successful MCP results for clients
that read `structuredContent`; it also controls help and startup error formatting.

Successful responses go to stdout. Errors go to stderr and cause a nonzero exit
code; in JSON mode, errors have an `error` string. Directory existence is left to
the commands that access it; help and version do not require a collection.
Doctor writes document diagnostics to stdout and exits 1 when it finds problems.

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

Initialize a collection with an example application data schema:

```sh
mise means init --path ./memory
```

By default, `init` creates the directory if needed and writes an example `schema.json`. It
refuses to overwrite an existing schema and leaves other files alone. Without
`--path`, it initializes the current directory. Collection loading uses exactly
the selected directory; it does not search parent directories.

Edit `schema.json` directly to define the fields inside `frontmatter.data`. The
file is a standalone JSON Schema; do not wrap its rules in a `data` property:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "tags": { "type": "array", "items": { "type": "string" } }
  },
  "additionalProperties": false
}
```

The starter schema accepts optional `tags` and rejects other data fields. An empty
`data` object is valid. Means separately requires a generated `id`, a nonempty
string `title`, and an object `data` in stored frontmatter. The top level is
reserved for means; application fields belong inside `data`. This also allows
application fields named `id`, `title`, or future means fields without collisions.

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
schema edits take effect immediately. Use `means doctor` to check existing
Markdown documents against the current schema.

## Documents

Create a document from a JSON envelope containing application metadata and a
Markdown body. `--input` accepts a file path or `-` for stdin:

```sh
mise means create --path ./memory --input - --json <<'JSON'
{
  "frontmatter": {
    "title": "Use Go for means",
    "data": {"tags": ["decision"]}
  },
  "body": "We chose Go for single-binary distribution.\n"
}
JSON
```

This example uses the default data schema. Adapt `frontmatter.data` to your
collection's `schema.json`. `frontmatter` and its nonempty `title` are required;
`data` defaults to `{}` and `body` to `""` when omitted. Explicit null is rejected
for either. Other envelope and means-owned frontmatter fields are rejected. Do not
supply `id` on creation. Input file paths resolve against the current working
directory, independently of `--path`.

Creation generates a collection-scoped 5-character random ID using lowercase
letters and digits (32 symbols, giving 33,554,432 candidates). Collisions retry;
existing IDs retain their original length. It writes `<id>-<slug-generated-from-title>.md`, for example
`k7m4p-use-go-for-means.md`. The slug retains lowercase Unicode letters and
digits, separates words with hyphens, and is bounded to 80 bytes; a title without
letters or digits uses `document`. Filenames are storage details. Consumers use
the stored `id`, which survives filename changes and moves into subdirectories.
Title updates leave the filename unchanged.

Generated IDs are checked across all visible document paths. Creation briefly
reserves a candidate with an exclusive `.means-create-<id>.tmp` file, preventing
concurrent means creators from allocating the same ID with different filenames.
Collisions generate another candidate. Allocation files are removed on completion;
a crash can leave one behind, but a future collision skips it without requiring
manual recovery. Existing files are never overwritten. Direct edits and Git merges
can still introduce duplicates: read/update refuse ambiguous IDs, and list reports
them even when a result limit hides the other matching document.

The stored file is ordinary Markdown:

```markdown
---
id: k7m4p
title: Use Go for means
data:
  tags:
    - decision
---
We chose Go for single-binary distribution.
```

A successful create is not idempotent: repeating it creates another document.

With `--json`, creation, reading, and updates return `{ "id": "...", "hash": "...",
"frontmatter": {...}, "body": "..." }`. The hash is lowercase SHA-256 of the exact
stored Markdown bytes, including frontmatter, whitespace, and line endings.
Text-mode creation/update prints just the ID; text-mode reading prints
the exact stored Markdown. Body content and trailing newlines are preserved.

```sh
mise means read <id> --path ./memory --json
mise means list --path ./memory --json
mise means list --path ./memory --limit 0 --json
```

Listing returns `{ "documents": [{ "id": "...", "frontmatter": {...} }],
"has_more": false }`, without bodies or hashes. Results are ordered by filename. The default
limit is 20; `--limit 0` returns all documents. Increase the limit when `has_more`
is true. Text output contains one ID and its metadata per line.

Search titles and Markdown bodies with a case-insensitive literal phrase. Repeat
`--filter KEY=VALUE` to match fields directly under `frontmatter.data`; filters
combine with AND. For this repository's issue collection, for example:

```sh
mise means search --path ./issues --filter priority=normal --filter type=feature --json
mise means search schema --path ./issues --limit 0 --json
```

Values that are not valid JSON, such as `normal`, are strings. JSON literals
keep their types: use `--filter count=2`, `--filter active=true`, or
`--filter 'tags=["search"]'`.
Nested objects match their supplied keys recursively; arrays must match in full.
A missing key does not match, including when the filter value is `null`. The
optional text query matches a phrase in either the title or body, not a regular
expression. Omit it for data-only filtering. Results are in relative filename
order and include ID, frontmatter, and a match field/excerpt for text searches,
without complete bodies or hashes. Use `read` for the full document.

Search defaults to 20 matching documents; `--limit 0` returns all. JSON output
contains `documents`, `has_more`, and `diagnostics`. Malformed files and duplicate
IDs are excluded from matches and reported in diagnostics regardless of the
query or limit. Schema-invalid application data remains searchable. Search
reads files directly without an index or any writes. Per-document diagnostics
retain exit code 0; invalid arguments or collection-level failures exit 1.

Manually created `.md` files work too when they contain the means envelope.
Stored IDs may contain 1–128 letters, digits, hyphens, or underscores, starting
with a letter or digit. Existing longer IDs remain supported. Discovery scans
visible subdirectories recursively and skips symlinks and hidden files/directories.
The root `instructions.md` contains collection guidance and is excluded even if
it has frontmatter. Reads accept IDs, not paths or filename stems.

Reads and lists check the means envelope without enforcing the current data
schema, so application data remains retrievable after schema changes. A valid
collection schema is still required. Malformed documents and duplicate IDs appear
in list results with an `error`, a relative diagnostic `path`, and null
`frontmatter`. Their `id` is empty if it cannot be extracted. These entries count
toward the limit and retain the successful list exit code. Reading an invalid or
ambiguous document fails rather than selecting a file arbitrarily.

Frontmatter uses string keys and JSON-compatible values. Duplicate keys, aliases,
merge keys, custom YAML tags, and non-finite numbers are rejected. Plain YAML dates
are returned as strings. Both LF and CRLF delimiters are supported.

Run `mise means doctor --path ./memory --json` to inspect every visible document
against the current schema. Doctor reports malformed Markdown/frontmatter, invalid
means envelopes, duplicate IDs, and application data that fails schema validation.
It uses the same file discovery rules as list and never changes documents. JSON
output is `{ "healthy": true, "checked": 0, "diagnostics": [] }` for an empty
collection. An unhealthy result contains one or more diagnostics with `id`,
relative `path`, and `error`; `id` is empty when it cannot be read. Healthy
collections exit 0. Unhealthy collections write the complete report to stdout
and exit 1 without a stderr error response. Collection-level errors, such as an
invalid `schema.json`, go to stderr and exit 1.

Run `mise means create --help`, `mise means read --help`, `mise means list
--help`, or `mise means search --help` for full input, output, and error details.
Help is also available as JSON.

### Migrating the earlier document format

Earlier versions used filename stems as IDs and flat application frontmatter.
Those files need explicit migration: preserve the previous filename stem as
frontmatter `id`, retain the document `title`, move all remaining application
fields under `data`, and remove `title` from the application schema's properties
and required list. Add a title if the previous schema did not provide one. A
previous application field named `id` belongs under `data.id`; it does not replace
the document ID. Bodies can remain byte-for-byte unchanged. Filenames may be
renamed to the new convention, but lookup uses the stored ID.

The repository's issue collection uses five-character IDs and canonical
`<id>-<title-slug>.md` filenames. Its earlier long IDs were explicitly regenerated
during cleanup. There is no automatic migration or legacy fallback.

## Patch updates

Read the current document, then supply its `hash` with only the fields to change:

```sh
mise means read <id> --path ./memory --json
mise means update <id> --path ./memory --input - --json <<'JSON'
{
  "hash": "<hash from read>",
  "frontmatter": { "title": "Corrected title", "data": {"tags": ["decision"]} }
}
JSON
```

`--input FILE` also works. Input requires a 64-character lowercase hexadecimal
`hash` and at least one of `frontmatter` or `body`; other fields are rejected.

Frontmatter uses [JSON Merge Patch (RFC 7396)](https://www.rfc-editor.org/rfc/rfc7396):
omitted keys stay unchanged, nested objects merge recursively, null removes a key,
and arrays replace as a whole. For example, `{"frontmatter":{"data":{"tags":null}}}`
removes tags. This patch format cannot set a literal null metadata field.
`frontmatter` itself must be an object, not null. The optional `body` is a complete
replacement string: omit it to retain the existing body, or use `""` to clear it.
`body:null` is rejected. An empty frontmatter patch is allowed.

Only `title` and `data` may be patched in frontmatter. Any supplied `id`, including
null or its current value, is rejected. The resulting title must remain nonempty
and data must remain an object. Updates preserve IDs and filenames and validate
the means envelope plus resulting data against the collection schema. A missing, malformed, or stale hash prevents the update. On conflict,
read again, review the intervening changes, and rebuild the patch using the new
hash. There is no force option. Successful results return the new hash; replaying
a content-changing update with its old hash fails. A no-op retains the exact
original bytes and hash but still checks the hash and schema.

Body-only edits preserve the existing frontmatter text. Metadata edits may
reformat YAML and remove comments. Unspecified body bytes stay unchanged.
Updates preserve file permission bits and write a same-directory temporary file
before replacing the document by rename, which is atomic on supported filesystems.

Updates check the hash when reading the document and again just before replacement.
Updates do not lock documents: a simultaneous write between the final check and rename can
still be overwritten. Temporary files are removed on normal completion; a crash
can leave one behind.

`means update --help` documents the full contract in text or JSON.

## MCP

Launch a server bound to one existing collection:

```sh
means mcp --path /absolute/path/to/collection
# From this checkout:
mise means mcp --path ./issues
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
| `search` | Optional `query`, `filter` object, `limit` (20 by default, 0 for all) | Matching documents with excerpts, diagnostics, and `has_more` |
| `read` | `id` | ID, hash, frontmatter, body |
| `create` | Required `frontmatter`, optional `body` | Created ID, hash, frontmatter, body |
| `update` | `id`, `hash`, and `frontmatter` and/or `body` | Updated ID, hash, frontmatter, body |

All tools use the configured collection; they do not accept arbitrary paths.
Call `info` before creating documents to learn the schema for `frontmatter.data`.
The server reloads the schema on every tool call, so edits take effect immediately.
Validation, IDs, listing limits, malformed-file handling, and number preservation
match the CLI. Successful creation is not idempotent.
MCP updates use the same patch rules and required content hash as the CLI. Call
`read` first, and refresh it on a conflict. Update is annotated as a write that
can replace or remove existing content.

Results contain `structuredContent` and, by default, an equivalent compact JSON
text block for clients that only read `content`. Use `means mcp --json` when every
client of that registration reads `structuredContent`; successful results then
have an empty `content` array. This repository's `issues-mcp` task
uses that mode. Tool failures still contain explanatory text and return MCP
`isError` results without ending the server.
Stdout contains only compact, newline-delimited MCP messages. Diagnostics and
startup failures go to stderr. `--json` never changes protocol framing.

Initialize collections and edit schemas/instructions through the CLI/filesystem;
these are not MCP tools yet. The server does not load `AGENTS.md` itself.
Run `means mcp --help` for the full contract.

### Collection identity and instructions

Give the schema a top-level `title` and `description` to tell agents what belongs
in this collection. These annotations describe the collection, separately from
application data fields inside `properties`:

```json
{
  "title": "Project issues",
  "description": "Bugs, features, and planned work for this project.",
  "type": "object",
  "properties": { "status": { "enum": ["todo", "done"] } },
  "required": ["status"]
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
usage guidance. Startup guidance is concise; `info` returns the complete means
usage guidance and the current collection workflow. It must be a regular file,
not a directory or symlink. Missing or empty instructions are allowed;
unreadable or invalid instructions fail MCP
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

Each registration exposes the same six tool names, bound to its own collection.
The client registration distinguishes the servers; schema annotations distinguish
their purpose, and each collection supplies its own workflow. No routing flag or
additional means configuration format is needed.

## Design direction

- **Plain files:** Markdown documents and collection schemas remain authoritative,
  editable, and suitable for version control.
- **Typed application data:** schemas describe `frontmatter.data` and can evolve through
  explicit consumer operations.
- **Single binary:** implemented in Go, with short-lived CLI commands and an
  optional MCP subprocess managed by its client.
- **Optional indexes:** means may persist derived data locally to support efficient
  retrieval. The storage format is undecided.
- **Diagnostics:** `doctor` identifies documents that do not satisfy their
  collection's schema. A possible `fix` command is future work.

The project's own backlog lives in the [issues](issues/) collection, including
search, deletion, and diagnostics work. Its [instructions](issues/instructions.md)
describe the workflow. List or filter the current issues from the repository root:

```sh
mise means list --path ./issues --limit 0 --json
mise means search --path ./issues --filter status=todo --filter priority=normal --json
```

Use `read`, `create`, and `update` with the same `--path` to manage this backlog.
[TODO.md](TODO.md) retains completed milestones; active work is tracked only in
`issues/`.

### Repo-local MCP

Root [.codex/config.toml](.codex/config.toml) registers this backlog as `issues`.
It launches `mise --quiet issues-mcp`, which fixes its working directory at the
repository root and serves `./issues` using the current Go source.

Start or restart Codex in this trusted checkout and use `/mcp` to check the
connection. Call `issues.info` to confirm the collection title is "means issues",
then use its `list`, `search`, `read`, `create`, and `update` tools. The collection supplies
its schema and workflow through MCP. The first startup may compile dependencies;
the configured timeout is 60 seconds. CLI access remains available as a fallback.

See [MCP configuration](https://learn.chatgpt.com/docs/extend/mcp?surface=cli)
for client setup details.

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
cd issues
mise means list --json
```

The entry point is `main.go`; command definitions and output formatting live in
`internal/cli`. Collection initialization, document storage, and schema validation live in
`internal/collection`, independently of the CLI. Development builds report `dev`
as their version. Release builds
can set it with `go build -ldflags "-X main.version=<version>"`.

See [AGENTS.md](AGENTS.md) for repository guidance.
