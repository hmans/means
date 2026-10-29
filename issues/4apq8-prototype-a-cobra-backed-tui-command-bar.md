---
data:
  priority: normal
  status: done
  tags:
    - tui
  type: feature
id: 4apq8
title: Prototype a Cobra-backed TUI command bar
---
Add a colon prompt with Cobra command dispatch, aliases, Bubbles text input, dim completion previews, and cycling view-name completion. Bare q must no longer quit. Keep scripting and collection-defined commands out of this initial prototype.

Implemented quit/q, views/view, filters/filter, clear, home, diagnostics, and help. Commands use fresh Cobra state; views supplies a dynamic argument completion callback. Tab and Shift+Tab cycle matches, view names are quoted automatically, errors remain in the prompt, and Esc cancels. Filters opens the existing picker; field/value command arguments are deferred. Verified with mise test, go vet, git diff --check, and a PTY test of completion rendering and quit execution.

Self-review: fixed completion for leading whitespace, escaped quotes and dash-prefixed view names; disabled destructive mid-line completion; made Tab accept the displayed suggestion; refreshed completion state after asynchronous clipboard input. Added regression tests for these cases, refresh preserving command input, and pane bounds. Updated documentation for the bordered command pane and error display. Full race suite, go vet, diff check, and PTY completion/error smoke checks passed.

Final behavior: home or Escape while browsing leaves the view and preserves filters; clear keeps the view and removes search and filters. The x keyboard shortcut clears filters, with no x command alias. Backspace on an empty command input cancels. View names replace column headings in the document pane. Picker and command view selection share focus behavior, invalid views have explicit errors, compact terminals show command errors, and help remains available while loading.
