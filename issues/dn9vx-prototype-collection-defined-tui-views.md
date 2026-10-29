---
data:
  priority: normal
  status: done
  tags:
    - cli
    - tui
  type: feature
id: dn9vx
title: Prototype collection-defined TUI views
---
Add an optional collection-local YAML app definition with named Expr-filtered views and field columns. Provide a searchable view picker, live config reload, useful errors, and a playable issues example; no editing or action execution.

Outcome: Added optional MEANS.app.yaml with named Expr boolean filters and direct field columns. A searchable, mouse-enabled Views picker retains All documents and combines selected views with text/data filters. Config edits reload live; syntax failures preserve the last working snapshot, while document evaluation errors appear in diagnostics. Filters expose id/title/body/data and a consistent now() per evaluation; numbers adapt to int64/float64 without changing collection metadata filter semantics. Added an issues app with Open work, Open bugs, TUI work, and guarded Recently created examples. Updated help, guide, and Sweet note. Validation: full mise test -race, go vet, diff check, regression tests for predicates/dates/numbers/config errors/live reload/mouse/layout, and PTY smoke testing of view selection. Prototype format and numeric/time reevaluation limits are documented; no actions or editing added.

View switcher refinement: Replace the full-pane picker with a dropdown anchored below Views. Keep documents and preview visible, mark and reveal the active view, retain type-ahead, and support keyboard paging, wheel scrolling, mouse selection, Escape, and outside-click dismissal without click-through. Bound the menu to terminal geometry and preserve background scrolling on cancel. Updated guide and feature note. Validation: TUI tests, go vet, diff check, PTY smoke test, and visual inspection of the dropdown; new tests cover menu bounds, active selection, dismissal, empty search, and scrolled mouse selection.

Configuration consolidation: Moved views into optional app.views in MEANS.md frontmatter and removed the separate issues/MEANS.app.yaml. Collection loading validates typed app/view fields, unknown keys, limits, titles, and column names; only the TUI compiles Expr expressions. The TUI uses the same loaded configuration snapshot as the schema. Schema replacement preserves app configuration and workflow bytes. Migrated the playable issues example and updated help, guide, agent instructions, and release note. Tests cover invalid app rejection before init writes, complete-input byte preservation, expression independence from collection operations, schema replacement, optional app removal, and the existing live reload/dropdown flows. Full race tests, go vet, diff checks, collection list smoke check, and PTY view switching passed.

Pre-PR cleanup: Invalid expressions now disable only their own views, with dropdown labels and diagnostics; All documents and valid siblings remain usable, and invalidating the selected view falls back to All documents. Reuse immutable compiled apps when configuration is unchanged. Evaluate custom views in background commands with generation checks to reject results superseded by selection, metadata filters, or snapshots; cancel work between documents and apply the latest text query on completion. Show an Updating view indicator while preserving navigation. Parse app configuration once per collection load, rename the diagnostic location constant, and extract dropdown rendering/geometry into tui_dropdown.go. Updated documentation and release note. Regression tests cover broken-view startup/recovery, cache reuse, stale results, filter changes, refresh, cancellation, and concurrent typing. Full mise test -race, go vet, diff checks, and PTY view-switching smoke tests passed.