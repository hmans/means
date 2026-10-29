---
data:
  priority: normal
  status: done
  tags:
    - tui
  type: feature
id: eb99p
title: Highlight hovered TUI documents and hide empty filters
---
Hide the empty data-filter row and highlight hovered documents with a subtle background, with a distinct selected background. Implemented for default and custom views using all-motion mouse reporting. Hover does not change selection or focus, follows scrolled content, and is suppressed during divider dragging and overlays. Verified with mise test, git diff --check, and a PTY mouse-motion smoke test with rendered screenshot inspection.