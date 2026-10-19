# means

An agentic memory substrate built from Markdown files.

means will provide a CLI for creating, reading, updating, searching, and deleting
documents in a directory. Each document has a Markdown body and typed
frontmatter, governed by a schema stored alongside the collection. Applications
and agents can establish and evolve that schema as their needs change.

**Status:** early development. The Cobra CLI scaffold provides help, a `version`
command, and shared flags. Document and schema operations are not implemented yet.

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

Schema syntax, document identity, and the initial command interface are still
being designed. See [TODO.md](TODO.md) for the next steps.

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
`internal/cli`. Development builds report `dev` as their version. Release builds
can set it with `go build -ldflags "-X main.version=<version>"`.

See [AGENTS.md](AGENTS.md) for repository guidance.
