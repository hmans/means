---
data:
  priority: normal
  status: done
  tags:
    - tui
  type: feature
id: ex58y
title: Show TUI data as YAML above the content
---
Display application data as syntax-highlighted YAML in its own bordered panel above Markdown content. Default to a one-third/two-thirds height split, independent scrolling, and a draggable horizontal separator. Preserve numeric precision, terminal bounds, and resize behavior.

Outcome: Added separate Data and Content panels with stable heights independent of document length. Each panel has its own scroll position and visible-range footer. The wheel scrolls the panel under the pointer without changing focus; clicking focuses it. Tab/Enter cycles Documents, Data, and Content, and Shift+Tab reverses direction. Paging uses the focused panel height. Dragging the separator or adjacent borders preserves at least one content row per panel; the chosen proportion persists across document selection and terminal resize for the session. Very short preview areas show only the focused panel. Rendered padding is omitted, and height changes reuse cached Markdown/YAML. Exact numbers, date-like strings, nested values, nulls, and multiline code fences retain their content. Updated the guide, command help, and consolidated fix release note.

Validation: Full mise test -race, mise exec -- go vet ./..., and git diff --check passed. Preview/TUI tests and vet passed again after the final focus-change scroll clamp. Regression coverage includes the default split, stable heights across document selection, independent mouse/keyboard scrolling and refresh retention, focused paging, drag capture/motion/release, minimum sizes, resize retention, overlay exclusion, YAML precision/highlighting, NO_COLOR, large metadata, and narrow layouts. Live PTY smoke testing confirmed the initial split, an SGR mouse drag from one-third to equal heights, content wheel/End scrolling without moving Data, and quit.

Layout refinement: Removed the blank row between Data and Content. Their borders now touch, and both separator borders remain draggable. Preview/TUI tests and git diff --check passed.