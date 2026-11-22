---
data:
  priority: normal
  status: done
  tags:
    - cli
    - mcp
    - views
  type: feature
id: 73zjq
title: Apply configured views from CLI and MCP list/search
---
MEANS.md `app.views` define named Expr filters, but only the TUI can use them. Since mmed0, CLI and MCP evaluate the same Expr language through `--where`/`where`, so a filter written once in a view must be retyped elsewhere.

Proposal: `means list --view "Open work"` and `means search <query> --view ...`, plus an MCP `view` field, applying the named view's filter.

Acceptance criteria:
- Looks up views by exact title; unknown titles fail and list the available views.
- ANDs the view filter with `--where`, `--filter`, and datetime flags, before limits.
- View errors follow `--where` semantics (fail the call), not the TUI's skip-and-report behavior; document the difference or align them.
- Decide whether view `columns` affect CLI table output or are TUI-only.
- MCP `info` (or a separate listing) exposes view titles so agents can discover them.
- Views remain marked experimental unless this issue promotes them.

## Outcome

Implemented in the collection package as `ListOptions.View`/`SearchOptions.View` (resolved via `AppConfig.View`), exposed as CLI `--view TITLE` on list/search and MCP `view`. The view filter ANDs with where/filter/datetime/query before sort, offset, limit, and has_more in all search modes. Errors follow `--where` semantics (compile and runtime errors fail the call); the guide documents that the TUI instead disables invalid views and skips failing documents. Columns stay TUI-only. CLI `info --json` and MCP `info` now return `views` (title, filter, columns). Views remain experimental; no view sort field was added.
