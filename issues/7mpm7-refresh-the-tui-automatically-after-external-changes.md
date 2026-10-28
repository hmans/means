---
data:
  priority: normal
  status: done
  tags:
    - cli
    - tui
  type: feature
id: 7mpm7
title: Refresh the TUI automatically after external changes
---
Check for external collection changes while the TUI runs. Preserve active search, filters, selection, picker input, and scroll state; handle additions, edits, removals, schema changes, and recoverable errors without blocking input.

Outcome: Added filesystem notifications through fsnotify with a 200 ms trailing debounce, replacing periodic scans. Watch visible collection directories and nested boundary directories, reconcile watches after directory changes, ignore hidden and nested collection content events, and support atomic editor replacements. Watch the parent for root replacement and close watches on TUI exit. Background refreshes preserve active input, filters, selection, and scroll state. Events during a load queue one follow-up; scans never overlap. Load failures retain the last usable snapshot and recover on subsequent changes or manual refresh. Updated CLI help, guide, and Sweet note. Validation: full mise test -race, go vet, and git diff --check passed. Actual filesystem tests cover write bursts, atomic replacement, deletion, new directories, hidden files, nested boundaries, and watch shutdown. Model tests cover unchanged snapshots, updates, active search, picker counts, recovery, schema changes, and changes during a load.