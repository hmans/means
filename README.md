# means

means is a general-purpose tool for managing collections of Markdown documents
with structured data and schemas. Use it for issue tracking, research notes,
agent memory, or other document-based workflows.

A collection is a directory of documents and a `MEANS.md` containing collection identity,
workflow guidance, and a JSON Schema expressed in YAML frontmatter. The files are the source of truth: people and other tools can edit them
directly.

Each document has a Markdown body and YAML frontmatter. Means generates its ID;
the caller supplies its title. Application fields live under `data` and are checked against the
collection's JSON Schema. Agents and applications can use the CLI with JSON
input and output, or connect to one collection through the optional stdio MCP
server. Ordinary commands run to completion; no daemon is needed.

means is in early development. It currently supports collection initialization,
validated schema replacement, creating and reading documents, hash-checked
updates and deletion, listing, ranked full-text search with data filters,
literal search, terminal tables, Markdown-rendered document viewing with `show`,
an interactive document browser with `tui`,
collection diagnostics, and opt-in OpenAI embeddings with semantic search.
Means creates a commented user config at `~/.config/means/config.yaml` (or under
`XDG_CONFIG_HOME`); edit it to enable embeddings on this machine.

## Try it

Homebrew publishing is configured for `hmans/homebrew-tap`. After the first
release through this workflow, install with `brew install --cask hmans/tap/means`.
See [release setup](docs/releases.md) for publishing requirements.

Run `means help init` for collection setup and `means help mcp` for MCP integration.
The same command help serves people, scripts, and agents.
Run `means help agents` for a built-in guide to collection structure and writing
`MEANS.md`, matched to the installed version.

From this checkout, use the Go toolchain configured by [mise](https://mise.jdx.dev/):

```sh
mise install
mise means init --path ./memory
mise means create --path ./memory --input - --json <<'JSON'
{"frontmatter":{"title":"Use Go for means","data":{"tags":["decision"]}},"body":"We chose Go for single-binary distribution.\n"}
JSON
mise means list --path ./memory --json
```

Run `mise deploy` to build this checkout and install `means` into `~/.local/bin`.
Add that directory to your `PATH` to run `means` from any directory. Run
`mise deploy` again to replace the installed binary with the current checkout.

`init` writes `MEANS.md` with a starter schema that accepts optional `tags`.
Edit it to fit your application, or supply a complete file with `init --input FILE`.
Use `init --schema FILE` to supply only a JSON Schema. Use
the ID returned by `create` to read and update a document; filenames can change
without changing identity.

See the [usage guide](docs/guide.md) for collection layout, commands, updates,
search, MCP setup, and migration notes. `mise means` shows a banner and available
commands. `mise means help` gives a short introduction.
`mise means help <command>` and `mise means <command> --help` give the same usage
help; `--json` changes the format, not the content.
Use `means info` to inspect a collection and `means edit <id>` to edit a document
with `$VISUAL` or `$EDITOR`.
Run `means tui --path ./issues` to browse documents, preview Markdown, and filter data.

## This repository

means tracks its own work in the [issues collection](issues/), using its
[configuration and workflow](issues/MEANS.md). Contributors
should read [AGENTS.md](AGENTS.md). Run `mise test` for the Go test suite.
See [releasing means](docs/releases.md) to record changes and prepare releases.
