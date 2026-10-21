# Using means

This guide covers the file format and common workflows. For exact flags, input
shapes, responses, errors, and limitations, run `means <command> --help` (or add
`--json` for machine-readable help).

## Collections

A collection is one directory with a standalone `schema.json`. The schema
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
existing `schema.json`. Edit the file directly to evolve a collection. Means
loads the selected directory exactly; it does not search parent directories.
Schema edits take effect on the next command or MCP call. Run `doctor` after a
schema change to find existing documents that need attention.

An optional root `instructions.md` provides collection workflow guidance to MCP
clients. It is configuration, not a document, and is excluded from lists and
reads. The server does not load `AGENTS.md`.

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
can introduce duplicate IDs; reads and updates reject ambiguous IDs, while
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
```

`list` returns IDs and frontmatter without bodies or hashes. Its default limit
is 20; `--limit 0` returns all. Malformed documents and duplicate IDs appear as
entries with errors and paths. `search` matches a case-insensitive literal phrase
in titles or bodies and/or fields under `data`. Repeated CLI `--filter KEY=VALUE`
flags combine with AND; JSON values retain their types. Search returns metadata
and excerpts, not full bodies or hashes, and reports malformed files separately.
Both commands read files directly without an index.

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

## MCP

`means mcp --path /absolute/collection/path` serves one collection over stdio.
The client starts and owns the subprocess; there is no network listener or
managed daemon. Register the command once per collection to give one agent
multiple collections. Each server advertises its collection identity and the
optional `instructions.md` workflow at startup. Call `info` for the current
schema and guidance. Other tools are `list`, `search`, `read`, `create`, and
`update`; they use the same collection behavior as the CLI. Each call reloads
the schema. Restart the server after changing collection title, description, or
workflow to refresh advertised discovery metadata.

MCP messages use protocol JSON framing regardless of `--json`. Successful tool
results include structured content and, by default, a compact JSON text copy
for clients that only read text. `means mcp --json` omits that text copy for
clients that read structured content; tool errors still contain explanatory
text. The flag also formats CLI help and startup errors. This repository's
[MCP registration](../.codex/config.toml) serves its
[issues collection](../issues/) through the `issues-mcp` mise task.

## Migrating earlier documents

Earlier versions used filename stems as IDs and flat application frontmatter.
Migrate those files explicitly: preserve the old filename stem as
`frontmatter.id`, retain or add `title`, move application fields under `data`,
and remove `title` from the application's schema properties and required list.
An application field named `id` belongs under `data.id`; it does not replace
the document ID. Bodies can stay unchanged. There is no automatic migration or
legacy fallback.
