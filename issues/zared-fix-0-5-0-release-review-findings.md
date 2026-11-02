---
data:
  priority: high
  status: done
  tags:
    - cli
    - search
    - releases
  type: bug
id: zared
title: Fix 0.5.0 release review findings
---
Apply the release review fixes: quote numeric editor fixture IDs; preserve JSON errors after malformed flag values; reject unknown schema/instructions subcommands; use exact array filter equality, including nested objects and precise numbers. Standardize public CLI examples on show --json and reconcile completed release automation work. Verify regression tests, race tests, vet, and all release snapshot targets before opening the PR.

Completed all four fixes, added regression coverage, standardized CLI examples on show --json, updated Homebrew installation wording, and closed nyd3k/qbt4w after checking the successful live v0.4.2 release workflow and published tap cask. Added one consolidated Sweet fix note.

Validation: mise test; mise test --race; go vet ./... through mise; 100 repetitions of TestEditorWorkflow/success; release:snapshot with all six platform archives and the Homebrew cask; git diff --check. All passed. No live embedding API calls or manual TUI checks were needed for these changes.
