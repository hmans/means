# Two brains for one agent

This example connects one agent to two means collections. Their schemas describe
what belongs in each, and their instructions supply the workflow through MCP.
The sample documents concern a fictional application called Meadow.

| MCP registration | Collection | Purpose |
| --- | --- | --- |
| `issues` | [issues/](issues/) | Bugs, features, planned work, and progress |
| `memory` | [memory/](memory/) | Durable facts, preferences, context, and notes |

Memory is deliberately simple: a generated `id`, required `title`, optional `data.tags`, and a free-form
Markdown body. It has no required categories or task-management fields.

## Use with Codex

Install the repository's Go toolchain with `mise install`, then start Codex here:

```sh
cd example
codex
```

The project-local [.codex/config.toml](.codex/config.toml) registers both servers.
Codex loads project configuration for trusted projects; trust this checkout
through Codex's normal project setup if prompted. Use `/mcp` to check that
`issues` and `memory` connected. Then try:

> What is the example application called, and what tone should its interface use?

> List the example issues and suggest which one to work on next.

> Remember that Meadow's example team prefers weekly written updates.

Each registration exposes `info`, `list`, `read`, `create`, and `update`, bound to its own
collection. The collection's title and purpose appear in every tool description.
[issues/instructions.md](issues/instructions.md) and
[memory/instructions.md](memory/instructions.md) provide their workflows through
server instructions and `info`. This example has no local `AGENTS.md`; collection
purpose and workflow come from MCP. For updates, read first and pass the returned
content hash with a patch; stale writes fail and require a fresh read. Means owns
`id` and `title`; each collection's schema validates the object under `data`.
Put issue fields such as status and priority under `data`. IDs are stored in
frontmatter and remain valid when filenames change.

The configuration runs `mise --quiet example-mcp` for issues and
`mise --quiet example-memory-mcp` for memory, using the current repository source.
These tasks fix their collection paths regardless of the client's working
directory within this checkout. The first launch may compile Go dependencies;
allow up to 60 seconds for startup. Starting Codex inside `issues/` or `memory/`
also loads both registrations. Your global Codex configuration is unchanged.

Restart the client/server after changing Go code or to refresh its advertised
title, tool descriptions, and connection instructions. Document/schema edits take
effect on the next tool call; `info` also reads current instructions.

To check either configured command manually, run it from anywhere in this checkout:

```sh
mise --quiet example-mcp
# Or, separately:
mise --quiet example-memory-mcp
```

Each waits for MCP messages on stdin. A blank terminal is expected; use Ctrl-C to
stop.

## CLI fallback

If MCP is unavailable, read the selected collection's `schema.json` and
`instructions.md` directly for guidance. From the repository root:

```sh
mise means list --path ./example/issues --limit 0 --json
mise means list --path ./example/memory --limit 0 --json
mise means read <id> --path ./example/memory --json
mise means create --path ./example/memory --input - --json
mise means update <id> --path ./example/memory --input - --json
```

Inside either collection directory, omit `--path`. The mise task preserves your
working directory, and relative input paths resolve there too. Use
`mise means <command> --help` for input and output details. Parse the whole
pretty-printed JSON response; failures exit 1 and return a JSON `error` on stderr.
Both collections are already initialized; do not run `init` over them.

Configuration follows the [Codex MCP documentation](https://developers.openai.com/codex/mcp).
