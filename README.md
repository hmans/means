# means

An agentic memory substrate built from Markdown files.

means will provide a CLI for creating, reading, updating, searching, and deleting
documents in a directory. Each document has a Markdown body and typed
frontmatter, governed by a schema stored alongside the collection. Applications
and agents can establish and evolve that schema as their needs change.

**Status:** early development. Collection initialization, schema loading, help,
and version reporting are implemented. Document operations are next.

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
can define their own fields. Markdown bodies are free-form, and documents will use
YAML frontmatter when document operations are implemented.

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

Document identity and the document command interface are still being designed.
See [TODO.md](TODO.md) for the next steps.

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
`internal/cli`. Collection initialization and schema validation live in
`internal/collection`, independently of the CLI. Development builds report `dev`
as their version. Release builds
can set it with `go build -ldflags "-X main.version=<version>"`.

See [AGENTS.md](AGENTS.md) for repository guidance.
