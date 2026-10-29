---
data:
  priority: normal
  status: done
  tags:
    - docs
  type: task
id: 3pykq
title: Polish README before Homebrew release
---
Tighten the README around purpose, current capabilities, installation, and a short first-use example. Keep detailed command and release contracts in their existing guides.

Outcome: Replaced the dense feature paragraph with a short list, separated installation from first use, and replaced the JSON-heavy example with init/create/list/tui commands. Removed low-level configuration and release-process details; retained guide links and an accurate pending-Homebrew-release note. Omitted a repository-maintenance release note at user request. Also removed tagline notes and consolidated TUI polish into the main user-facing feature notes; preserved breaking-change migration details. Sweet status still reports 0.4.0.

Validation: Built the binary and ran init/create/list in a temporary collection with isolated user configuration. Verified tui help and all local README links. git diff --check passed. Interactive TUI and Homebrew installation were not run.