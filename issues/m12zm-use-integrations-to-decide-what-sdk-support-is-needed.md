---
data:
  priority: normal
  status: done
  tags:
    - sdk
    - design
  type: task
id: m12zm
title: Use integrations to decide what SDK support is needed
---
Go consumers need in-process access to means collections without starting the CLI or speaking MCP. The user chose to expose the existing collection implementation as a public Go package in the same module.

Acceptance criteria:
- Record integration needs that CLI or MCP do not adequately cover.
- Decide whether to provide an SDK, its language and scope, or defer it.
- Preserve ordinary CLI commands as short-lived processes.

Decision and outcome:
- Provide github.com/hmans/means/collection for Go consumers. Keep one module and release version, and retain the root CLI entry point and installation path.
- CLI, TUI, and MCP use the public package. Expose typed creation/options, explicit operation contexts, schema and workflow updates, diagnostics, and programmatic errors.
- Keep user config and .env loading in the host. Supply opt-in embedding settings explicitly. Loaded collection configuration is an immutable snapshot; reload after configuration changes.
- Preserve document formats, stable IDs, hash checks, cache behavior, and the existing external-write race limits.
- Add the Go integration guide and executable examples. Keep dynamic application data maps; application-specific typed models remain future integration work.

Validation:
- mise test -race and mise exec -- go vet ./... passed, including the existing CLI/MCP and collection tests.
- A separate consumer module imports the public package and exercises document operations without a CLI subprocess.
- Public API tests cover input precision, diagnostics, stale hashes, configuration snapshots, copy isolation, embedding defaults, and cancellation.
- Regression tests first reproduced ignored search-lock deadlines and inconsistent native Go values in validation/list/search; both pass after the fixes.

Migrated from TODO.md.
