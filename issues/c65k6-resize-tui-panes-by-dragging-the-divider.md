---
data:
  priority: normal
  status: done
  tags:
    - tui
  type: feature
id: c65k6
title: Resize TUI panes by dragging the divider
---
Allow dragging the gap and adjacent borders between the document list and preview. Keep minimum pane widths, reflow the preview, and retain the preferred width for the session.

Implemented mouse drag and release handling, minimum widths of 32 columns, preview reflow, and session width retention across terminal resize. Menus do not start a divider drag. Verified with mise test, git diff --check, and a PTY smoke test using SGR mouse press, motion, and release events; inspected the rendered result.