---
data:
  priority: high
  status: todo
  tags:
    - search
    - cli
    - mcp
  type: feature
id: mmed0
title: Use Expr filters in CLI and MCP, not only TUI views
---
MEANS.md views already use Expr (`data.status in ["todo", "in_progress"]`), but Expr is compiled only in the TUI (`internal/cli/tui_app.go`). CLI and MCP offer exact `--filter KEY=VALUE` plus four datetime operators. Two filter languages, and the non-TUI one cannot express "in", "not", ranges, or nested conditions.

Proposal: move Expr compilation/evaluation into the collection package and add `--where '<expr>'` to list/search, plus a `where` field in MCP.

Acceptance criteria:
- One shared Expr environment (same variables/functions as views) used by TUI, CLI, and MCP.
- Applied before limits/offsets/has_more, combined with existing filters using AND.
- Invalid expressions fail with clear errors.
- Decide whether datetime flags become redundant and can be deprecated.
