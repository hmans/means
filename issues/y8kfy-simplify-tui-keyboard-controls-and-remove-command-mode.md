---
data:
  priority: normal
  status: done
  tags:
    - tui
  type: task
id: y8kfy
title: Simplify TUI keyboard controls and remove command mode
---
Remove the experimental colon command bar and Vim-style navigation. Restore direct keyboard shortcuts, including q to quit outside text input, while retaining arrow navigation, search, filters, views, and mouse controls. Update help, guide, release notes, and relevant tests. Supersedes the command-bar prototype in 4apq8.

Completed: removed command input, completion, command dispatch, command-specific rendering, and unused Bubbles/shlex dependencies. Removed j/k navigation; arrows, paging, Tab, direct action shortcuts, and mouse controls remain. Restored q to quit outside search and pickers, including while loading. Updated shortcut hints, CLI help, usage guide, and Sweet release note. Regression coverage verifies navigation, removed keys, and q behavior in browsing, diagnostics, loading, search, and all pickers; existing mouse tests now use arrow keys. Validation: mise test and git diff --check passed.