# means

means manages collections of Markdown documents with structured metadata.
Use it for issue tracking, research notes, agent memory, and other document-based
workflows. It ships as a single Go binary, with no managed daemon.

A collection is a directory of Markdown files with YAML frontmatter. Its
`MEANS.md` defines a JSON Schema for application data and guidance for working
with the collection. The files are the source of truth: edit them with means,
your editor, or other tools.

means is in early development. It includes:

- Document creation, editing, and deletion with stable IDs and stale-write checks.
- Full-text search, metadata filters, and opt-in semantic search.
- An interactive terminal browser with Markdown previews.
- Schema validation and collection diagnostics.
- JSON input/output and a stdio MCP server for applications and agents.
- A public Go package for direct collection access from other Go projects.

## Install

Install with [mise](https://mise.jdx.dev/) from GitHub Releases:

```sh
mise use -g github:hmans/means@latest
```

Follow mise's [shell setup](https://mise.jdx.dev/installing-mise.html#shells)
instructions to make `means` available on your `PATH`. Omit `-g` to add means to
the current project's mise configuration instead.

Or install with Homebrew:

```sh
brew install --cask hmans/tap/means
```

To build from this checkout, install [mise](https://mise.jdx.dev/), then run:

```sh
mise install
mise deploy
```

This installs `means` into `~/.local/bin`; add that directory to your `PATH`.

### Ask your agent

Copy either prompt into your coding agent:

```text
Please set up project memory using means!
```

```text
I want to plan this project using means.
```

## Try it

Create a collection and add your first note:

```sh
means init --path ./notes
means create --path ./notes --title "My first note"
means list --path ./notes
means tui --path ./notes
```

Edit `notes/MEANS.md` to define your collection's schema and workflow.
Use `means edit <id> --path ./notes` to open a note in your editor.

See the [usage guide](docs/guide.md) for collection design, search, and MCP setup.
Run `means help <command>` for command details, or `means help agents` for a
guide to using means from an agent.
For Go applications, see the [Go integration guide](docs/go.md).

## This repository

means tracks its own work in the [issues collection](issues/), using its
[configuration and workflow](issues/MEANS.md). Contributors
should read [AGENTS.md](AGENTS.md). Run `mise test` for the Go test suite.
See [releasing means](docs/releases.md) to record changes and prepare releases.

## License

Copyright 2026 the means contributors.

The project's source code, documentation, examples, and configuration are
licensed under the [Apache License, Version 2.0](LICENSE) (`Apache-2.0`).
