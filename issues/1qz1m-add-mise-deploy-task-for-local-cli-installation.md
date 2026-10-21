---
data:
  priority: normal
  status: done
  tags:
    - cli
  type: task
id: 1qz1m
title: Add mise deploy task for local CLI installation
---
Add a mise deploy task that builds the current checkout and installs means into ~/.local/bin with go install. Add a short README note. Verify mise deploy and the installed means --help command.

Outcome: Added the deploy task with the repository root as its working directory and GOBIN set to ~/.local/bin. Documented installation and replacement in README.md.

Validation: mise deploy succeeded; command -v means resolved to /Users/hmans/.local/bin/means; means --help succeeded; git diff --check passed. No Go source changed.