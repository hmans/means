---
data:
  priority: normal
  status: done
  tags:
    - tui
  type: bug
id: k1ksg
title: Refresh selected TUI document contents after external edits
---
The TUI detects document changes, but the selected document preview can retain old contents. Related work: 7mpm7 (filesystem refresh) and dn9vx (asynchronous collection views).

Acceptance: after an external body or metadata edit, refresh both preview panes even if search, navigation, or resize events render the old snapshot while a collection view is being evaluated. Preserve selection, active input, and scroll positions where possible. Add regression coverage.

Outcome: The preview cache now compares the full loaded document and pane width, instead of relying on an ID invalidated before asynchronous view evaluation. Intervening input or resize events can no longer cache old contents after the refreshed view arrives. Added regression cases for body, data, and title edits with search input and terminal resizing during view refresh. All three cases failed before the fix and pass after it. Updated the guide and added a fix release note.

Validation: mise test -race, go vet ./... through the mise toolchain, and git diff --check passed.