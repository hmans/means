---
data:
  priority: normal
  status: done
  tags:
    - cli
    - docs
  type: feature
id: mn257
title: Simplify CLI workflows and command discovery
---
Implemented the approved CLI review: info; create --title with optional --body-file; editor workflow with hash checks, current-schema validation and recovery copies; delete --hash; focused human command help and complete help agents <command>/JSON references; show --raw with hidden read compatibility; shared list/search metadata filters; grouped command help; side-effect-free informational commands. Existing structured inputs remain supported. Includes prior help split and general-purpose product framing.

Editor metadata uses full replacement, preserving literal nulls and removing omitted keys. VISUAL/EDITOR accepts an executable path; use wrapper scripts for arguments. Failed edits retain a private temporary file and report its path. Ordinary external-write race limitations still apply.

Validation: full mise test -race, mise exec -- go vet ./..., and git diff --check passed. Tests cover human/agent help, config side effects, info, list/search parity, raw output, create/delete alternatives, replacement validation and precision, editor success/failure/conflicts, and schema changes while editing. Manually inspected grouped banner, concise create help, full nested command reference, and info JSON. Documentation, project instructions, and Sweet release notes updated. Prepared as one PR.