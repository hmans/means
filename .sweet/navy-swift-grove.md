---
type: feature
---

Filter `list` and `search` with Expr expressions, the same language MEANS.md views use. Pass `--where 'data.status in ["todo", "in_progress"] and not ("tui" in (data.tags ?? []))'` on the CLI, or `where` to the MCP `list` and `search` tools. Expressions see `id`, `title`, `body`, `data`, and `now()`, combine with `--filter` and datetime flags using AND, and apply before offsets and limits in every search mode. Invalid expressions, evaluation errors, and non-boolean results fail the request with a clear error.
