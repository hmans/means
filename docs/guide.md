# Using means

This guide covers the file format and common workflows. For exact flags, input
shapes, responses, errors, and limitations, run `means <command> --help` (or add
`--json` for machine-readable help).

## Collections

A collection is one directory with a standalone `means.schema.json`. The schema
describes only `frontmatter.data`; means owns the document's top-level `id` and
`title`. Schemas use JSON Schema Draft 2020-12 by default. Internal references
work; references to other files or remote schemas are not supported.

Initialize with the starter schema or one you wrote:

```sh
mise means init --path ./memory
mise means init --path ./memory --schema ./memory.schema.json --json
mise means init --path ./memory --schema - --json < ./memory.schema.json
```

The starter schema accepts optional `tags` and rejects other application fields.
`init` validates a supplied schema before creating files and never overwrites an
existing `means.schema.json`. It also creates `.gitignore` with `.means/`, or
appends that rule while preserving existing content and avoiding duplicates.
The search cache is created only when first used. Use `schema set` to validate a replacement, or edit
the file directly to evolve a collection. Means
loads the selected directory exactly; it does not search parent directories.
Schema edits take effect on the next command or MCP call. Run `doctor` after a
schema change to find existing documents that need attention.

An optional root `MEANS.md` provides collection workflow guidance to MCP
clients. It is configuration, not a document, and is excluded from lists and
reads. The server does not load `AGENTS.md`.

### Changing the schema

Use `schema set` to check a proposed schema against existing documents before
installing it:

```sh
mise means schema set --path ./memory --input new-schema.json --json
mise means schema set --path ./memory --input - --force --json < new-schema.json
mise means doctor --path ./memory --json
```

Input is the JSON Schema itself. Means parses and compiles it with the same
rules as `init`; invalid or unsupported schemas always fail. The command requires
an existing regular `means.schema.json` file, but can replace an invalid schema.
It checks visible documents against the proposal, excluding nested collections.
This tests existing data, not abstract compatibility with the old schema or
future callers. A stricter schema can pass on an empty collection.

By default, data violations, malformed envelopes, and duplicate IDs prevent
installation. The command exits 1 and reports IDs, paths, and errors on stderr;
the schema stays unchanged. `--force` allows these document diagnostics and
installs the schema, returning exit 0 with the diagnostics on stdout. It never
bypasses schema validation or filesystem errors. JSON results include `installed`,
`healthy`, `checked`, and `diagnostics`; rejected compatibility checks also have
an `error` field. Other errors use the usual `{"error":"..."}` format.

Only the schema changes. Its supplied bytes and existing permission bits are
preserved through a same-directory temporary replacement. There are no locks:
concurrent document edits can invalidate the check, and schema edits can race
replacement. Observed schema changes abort installation.

After a forced change, the agent or user migrates documents: read each affected
ID, patch its data using the returned hash, and run `doctor` again. Reads and
searches remain available for schema-invalid data; each update must leave the
whole document valid. For staged migrations, first accept both old and new
forms, migrate documents, then tighten the schema. Means does not rewrite data
automatically. MCP calls load the new schema; restart MCP to refresh advertised
collection metadata. MCP provides the same operation through `schema_set`:

```json
{"schema":{"type":"object","required":["status"]},"force":false}
```

Pass a schema object or boolean, not a JSON string. `force` defaults to false.
Rejected compatibility checks return a tool error with structured diagnostics;
forced installations return success with those diagnostics. Error responses
retain explanatory text. Schema numbers retain
their precision, but JSON formatting is regenerated. A running server can repair
an externally invalidated schema; starting a server still requires a valid schema.

### Subdirectories and collection boundaries

Use the filenames `means.schema.json` and `MEANS.md` as written. Means uses
ordinary filesystem lookup; case sensitivity depends on the filesystem. It does
not search for other case variants. A `MEANS.md` file alone does not start a
collection, and an unrelated `schema.json` does not establish a boundary.

Ordinary subdirectories belong to the selected collection. A subdirectory with
its own `means.schema.json` starts a separate collection: all document operations stop
at that directory, including list, search, doctor, ID lookup, updates, deletion, and ID
allocation. The presence of the schema path establishes the boundary, even if
the schema is invalid, a directory, or a broken symlink.

For example, with `foo/means.schema.json` and `foo/bar/means.schema.json`, `--path foo`
excludes everything under `foo/bar`. Use `--path foo/bar` to work with that
collection. IDs are unique within each collection; parent and child collections
can reuse an ID. Schemas and `MEANS.md` guidance do not inherit from parents.

Adding or removing a nested `means.schema.json` changes which collection owns that
subtree on the next operation. Moving a document across a collection boundary
changes the schema that applies to it and can introduce an ID collision in the
destination. Moving it within one collection preserves its identity.

## Documents

Create a document by passing a JSON envelope from a file or stdin:

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

Means generates a random five-character ID and writes an
`<id>-<title-slug>.md` filename. Clients use the stored ID, not the filename.
Renames and moves into visible subdirectories preserve identity; title updates
leave filenames alone. Discovery ignores hidden paths and symlinks. Direct edits
can introduce duplicate IDs; reads, updates, and deletions reject ambiguous IDs, while
lists report each affected file. Creation checks for collisions and reserves a
candidate ID temporarily so concurrent means creators cannot allocate it twice.

On creation, `title` is required, omitted `data` defaults to `{}`, and omitted
`body` defaults to an empty string. Explicit `null` is invalid for either.
Application fields belong under `data`, including fields named `id` or `title`.
Creation and updates validate the data against the current schema. Reads and
lists check the means envelope but allow schema-invalid application data to be
retrieved after schema changes.

Creation is not idempotent: retrying a successful call creates another document.

## Finding and updating documents

```sh
mise means list --path ./memory --limit 0 --json
mise means search decision --path ./memory --json
mise means search --path ./memory --filter 'tags=["decision"]' --json
mise means read <id> --path ./memory --json
mise means show <id> --path ./memory
```

Without `--json`, `list` and `search` show colored tables of ID and title, followed
by diagnostics and limit notices. Tables wrap to the terminal width (80 columns
when piped). Piped output and `NO_COLOR` disable terminal styling. Use `--json`
for complete metadata and search excerpts.
Add `--with-body` to `list --json` or `search --json` to include each document's
Markdown body. Empty bodies appear as `"body": ""`; diagnostic entries have no
body. The flag requires `--json` and does not add hashes or change limits or order.

`show <id>` prints the title, ID, and Markdown-rendered body, including styled
headings, accented bullet markers, and code blocks. Prose wraps to at most 80
columns, shrinking to fit narrower terminals. `show --json` returns the same document and
content hash as `read --json`. Use `read` without `--json` for the exact source,
including YAML frontmatter. Neither command enforces the current application data
schema, so documents remain readable after schema changes.

`list --json` returns IDs and frontmatter without bodies or hashes. Its default limit
is 20; `--limit 0` returns all. Malformed documents and duplicate IDs appear as
entries with errors and paths. `search` uses full-text ranking by default: all
query terms must occur, in any order, across the title and body. Title matches
receive a boost; equal scores sort by stored ID. Queries use lowercase Unicode
tokens with no stemming, stop-word removal, or query-language operators.
For example, `search "deployment timeout"` finds "timeout during deployment".
A query with no tokens matches nothing.

Use `search "deployment timeout" --literal` for the original case-insensitive
substring search. Literal searches and searches with no query use filename order.
Repeated `--filter KEY=VALUE` flags combine with AND and apply before the result
limit. JSON numbers retain their precision, objects match recursively, and arrays
must match in full. Search JSON returns metadata and excerpts, not full bodies or hashes.
Full-text excerpts prefer the title or body passage containing the most distinct
query terms. Repeated copies of one term do not outweigh several different terms.
Ties prefer the title, then the earliest passage. Passages contain at most 180
Unicode characters plus ellipses; whitespace is collapsed. One excerpt cannot
always show terms that occur in different fields or far apart. Literal-mode
excerpts retain their original behavior.
Malformed documents and every occurrence of duplicate IDs are excluded and reported
as diagnostics, regardless of query, filter, or limit. Schema-invalid data remains
searchable. MCP offers the same behavior with the optional `literal` boolean.

### Search cache

Each collection stores a disposable Bleve index in `.means/search/`, for example
`issues/.means/search/`. Every full-text search scans and hashes the visible
documents, then updates changed index entries and removes absent or invalid ones.
Content hashes detect changes even when file size and timestamps stay the same.
Renames, deletions, and changes to nested collection boundaries are reflected in
the next search. Nested collections keep independent caches. Source documents
and schemas remain authoritative; create, update, and delete need no cache writes.

Full-text search also keeps parsed metadata in `.means/documents.jsonl`. Each
record contains a relative path, SHA-256 content hash, frontmatter, and the byte
offset where the body starts. Unchanged files reuse their parsed frontmatter;
bodies come from the source files. Every file is still read and hashed. Invalid
documents are parsed again so diagnostics stay current. Application data is not
validated against the schema during search.

The JSONL file starts with a format version and a checksum of the remaining
records. Missing, corrupt, truncated, or obsolete files are rebuilt. Refreshes
use a temporary file and atomic replacement under the search lock; unchanged
records skip both JSON encoding and writing. This is a disposable internal format, not an
interchange contract. Removing `.means/documents.jsonl` while no search is running
forces metadata to be parsed again. Other commands, including literal and
data-only searches, continue to read documents directly.

Search acquires `.means/search.lock`, opens and refreshes the index, queries it,
then closes it and releases the lock. Concurrent searches wait at most five
seconds before failing. OS locks are released when a process exits. MCP uses this
same per-call lifecycle. There is no background watcher or daemon. External editors
do not take the lock: results reflect files observed during the scan, not an atomic
filesystem snapshot. Concurrent edits may appear on the next search.

Missing indexes, obsolete index formats, and indexes that cannot be opened are
rebuilt. Permission failures and symlinked cache paths produce errors. To force
a rebuild, remove `.means/search/` while no search is running. Do not remove the
lock file while searches are active. Full-text search needs a writable cache;
literal and data-only searches use direct scans and create no cache. For older
collections, add `.means/` to their `.gitignore` manually.

### Updating documents

Read before updating. JSON read/create/update results include an ID, frontmatter,
body, and a SHA-256 hash of the exact stored file bytes. Supply that hash with
only the fields to change:

```sh
mise means update <id> --path ./memory --input - --json <<'JSON'
{
  "hash": "<hash from read>",
  "frontmatter": {"title": "Corrected title", "data": {"tags": ["decision"]}}
}
JSON
```

Metadata uses JSON Merge Patch: omitted keys remain, nested objects merge,
`null` deletes a key, and arrays replace. This patch format cannot set a literal
null metadata value. Supply `body` to replace the whole Markdown body; omit it
to keep the existing body or use `""` to clear it. IDs cannot be patched.
Updates validate the resulting document and reject stale hashes. On conflict,
read again and review the intervening changes before retrying. Updates use a
same-directory temporary replacement, but do not lock the document against a
write racing the final replacement.

`mise means doctor --path ./memory --json` checks every visible document against
the current schema without changing files. It reports malformed frontmatter,
invalid means fields, duplicate IDs, and application schema violations. Healthy
collections exit 0; an unhealthy report goes to stdout and exits 1.

CLI success goes to stdout; errors go to stderr. With `--json`, ordinary commands
return structured responses and errors such as `{"error":"..."}`. `--path`
defaults to the current working directory. Relative collection and input file
paths both resolve against the process working directory; an input path is not
relative to the selected collection.

## Deleting documents

Read the document, review it, then supply its ID and current hash:

```sh
mise means read <id> --path ./memory --json
mise means delete <id> --path ./memory --input - --json <<'JSON'
{"hash":"<hash from read>"}
JSON
```

Deletion permanently removes only the document file. It preserves directories,
collection configuration, and unrelated files. There is no trash, undo, force
option, or confirmation prompt. For archival, use an application field such as
`data.status` instead. The JSON result is `{"id":"..."}`; text output is the ID.

Missing or duplicate IDs, malformed document envelopes, and stale or invalid
hashes fail without deletion. A repeated deletion fails because the document is
missing. Data that fails the current application schema can still be deleted;
the selected collection must have a valid schema. Collection boundaries apply.
As with updates, there are no locks: a concurrent write between the hash check
and removal can still be deleted. On conflict, read again and review the changes.

## MCP

`means mcp --path /absolute/collection/path` serves one collection over stdio.
The client starts and owns the subprocess; there is no network listener or
managed daemon. Register the command once per collection to give one agent
multiple collections. Each server advertises its collection identity and the
optional `MEANS.md` workflow at startup. Call `info` for the current
schema and guidance. Other tools are `list`, `search`, `read`, `create`,
`update`, `delete`, and `schema_set`; they use the same collection behavior as the CLI. Each document call reloads
the schema. Restart the server after changing collection title, description, or
workflow to refresh advertised discovery metadata.

MCP messages use protocol JSON framing regardless of `--json`. Successful tool
results always use `structuredContent` without a redundant text copy; clients
must consume structured results. Tool errors still contain explanatory text.
`--json` only formats CLI help and startup errors; it does not change MCP results.
This repository's
[MCP registration](../.codex/config.toml) serves its
[issues collection](../issues/) by running `mise run --raw means -- mcp --path ./issues`
from the repository root. The MCP registration sets the working directory and
arguments directly; no collection-specific mise task is needed. Raw mode keeps
stdio connected to the MCP client. Each server startup runs the current source;
restart the server to pick up code changes.

## Migrating earlier collections

Rename each collection's `schema.json` to `means.schema.json`, including nested
collections, before using this version. There is no fallback to the old schema
filename. A nested collection with only the old filename is now an ordinary
subdirectory of its parent collection.

Rename the collection's root `instructions.md` to `MEANS.md` to keep its workflow
guidance available. Only `MEANS.md` is reserved for this purpose; the old name is
now treated as a document and must have valid frontmatter if kept. Restart MCP
servers after the rename to refresh their advertised workflow.

Earlier versions used filename stems as IDs and flat application frontmatter.
Migrate those files explicitly: preserve the old filename stem as
`frontmatter.id`, retain or add `title`, move application fields under `data`,
and remove `title` from the application's schema properties and required list.
An application field named `id` belongs under `data.id`; it does not replace
the document ID. Bodies can stay unchanged. There is no automatic migration or
legacy fallback.
