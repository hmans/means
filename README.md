# means

means is a small Go tool for agentic memory built from Markdown files. A collection
is a directory of documents and a `schema.json` that defines their application
data. The files are the source of truth: people and other tools can edit them
directly.

Each document has a Markdown body and YAML frontmatter. Means assigns its ID and
title; application fields live under `data` and are checked against the
collection's JSON Schema. Agents and applications can use the CLI with JSON
input and output, or connect to one collection through the optional stdio MCP
server. Ordinary commands run to completion; no daemon is needed.

means is in early development. It currently supports collection initialization,
creating and reading documents, hash-checked updates, listing, literal text and
data search, and collection diagnostics. Deletion is still being designed.

## Try it

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

`init` writes a starter schema that accepts optional `tags`. Edit `schema.json`
to fit your application, or supply your own schema during initialization. Use
the ID returned by `create` to read and update a document; filenames can change
without changing identity.

See the [usage guide](docs/guide.md) for collection layout, commands, updates,
search, MCP setup, and migration notes. `mise means --help` and
`mise means <command> --help` give the complete command contracts, including
JSON help.

## This repository

means tracks its own work in the [issues collection](issues/), using its
[schema](issues/schema.json) and [workflow](issues/MEANS.md). Contributors
should read [AGENTS.md](AGENTS.md). Run `mise test` for the Go test suite.
