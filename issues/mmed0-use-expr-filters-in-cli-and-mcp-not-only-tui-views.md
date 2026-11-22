---
data:
  priority: high
  status: done
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

## Outcome

Done. `collection.CompileExpression` / `(*Expression).Match` hold the single Expr environment (`id`, `title`, `body`, `data`, `now()`; int64/float64 number adaptation; node and memory limits). The TUI views now use it (duplicate code removed). `ListOptions.Where` and `SearchOptions.Where` apply it after exact and datetime filters (AND) and before offset/limit/score threshold/has_more in data-only, literal, full-text, and semantic modes. CLI: `--where` on `list` and `search`. MCP: `where` string on `list` and `search` (MCP list gained its first filter).

Error policy: strict. Compile errors fail with "invalid where expression"; runtime errors and non-boolean results on any candidate fail the call and name the document ("where expression failed for document ID"). Guard optional fields with `!= nil` / `??`. (TUI views keep their per-document diagnostics behavior.) `now()` is fixed per operation.

Datetime flags: kept. `--where 'date(data.at) > date("...")'` covers the same comparisons, but the semantics differ (datetime flags strictly require RFC 3339 with offset and skip invalid/missing values; Expr `date()` parses more formats and errors on invalid values). They could be deprecated later, after a release where `--where` is the documented path; not recommended to remove now since MCP clients use `datetime`.

Validation: new tests in `collection/expr_test.go` (all four search modes, list offset/limit/has_more, compile/runtime/non-boolean errors), CLI `TestListAndSearchWhere`, MCP stdio list/search `where`; `mise test` passes.
