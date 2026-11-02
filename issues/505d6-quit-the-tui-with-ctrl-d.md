---
data:
  priority: normal
  status: done
  tags:
    - tui
  type: feature
id: 505d6
title: Quit the TUI with Ctrl-D
---
Add Ctrl-D as an immediate quit shortcut throughout the TUI, including search, pickers, loading, and the command bar. Keep the existing quit shortcuts and update command help and the guide.

Outcome: Ctrl-D now uses the same immediate quit path as Ctrl-C in the browser and command bar. Updated CLI help, the guide, the small-terminal message, and a Sweet feature note. The browser handles the shortcut before the loading guard.

Validation: mise test and git diff --check passed. A real PTY smoke test confirmed successful Ctrl-D exit from documents, search, field and value pickers, views, diagnostics, and the command bar.