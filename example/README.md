# Issue tracker example

This collection demonstrates how a schema and collection instructions turn means
into an issue tracker. The sample issues describe a fictional application.

## Use with Codex

Install the repository's Go toolchain with `mise install`, then start Codex here:

```sh
cd example
codex
```

The project-local [.codex/config.toml](.codex/config.toml) configures an `issues`
MCP server. Codex loads project configuration for trusted projects; trust this
checkout through Codex's normal project setup if prompted. Use `/mcp` to check
that the server connected, then ask, for example:

> List the example issues and suggest which one to work on next.

[issues/schema.json](issues/schema.json) names this brain **Example issues** and
describes its purpose. That identity appears in every MCP tool description.
[issues/instructions.md](issues/instructions.md) supplies the issue workflow through
MCP server instructions and `collection_info`, so other clients receive it too.
This example has no local `AGENTS.md`; collection purpose and workflow come from MCP.
The server exposes collection information, listing,
reading, and creation. Updates still require editing the Markdown file directly.

The configuration runs `mise --quiet example-mcp`, using the current repository
source. That task fixes the collection path regardless of the client's working
directory within this checkout. The first launch may compile Go dependencies; allow up to
60 seconds for startup. It also works when starting Codex inside `issues/`.
It is scoped to this example and does not change your global Codex configuration.
Restart the client/server after changing Go code or to refresh its advertised
title, tool descriptions, and connection instructions. Document/schema edits take
effect on the next tool call; `collection_info` also reads current instructions.

To give an agent another means brain, register another `means mcp --path ...`
command under a distinct client name, such as `decisions`. Each registration has
the same operations but its own collection, purpose, and workflow. See the root
README for a multi-collection configuration example.

To check the configured command manually, run this from `example/issues/`:

```sh
mise --quiet example-mcp
```

It waits for MCP messages on stdin. A blank terminal is expected; use Ctrl-C to
stop.

## CLI fallback

If MCP is unavailable, use the CLI and read [issues/schema.json](issues/schema.json)
and [issues/instructions.md](issues/instructions.md) directly for guidance.
From `example/issues/`:

```sh
mise means list --limit 0 --json
mise means read <id> --json
mise means create --input - --json
```

From the repository root, add `--path ./example/issues`. The mise task preserves
your working directory, and relative input paths resolve there too. Use
`mise means <command> --help` for input and output details. Parse the whole
pretty-printed JSON response; failures exit 1 and return a JSON `error` on stderr.
The example is already initialized; do not run `init` over it.

Configuration follows the [Codex MCP documentation](https://developers.openai.com/codex/mcp).
