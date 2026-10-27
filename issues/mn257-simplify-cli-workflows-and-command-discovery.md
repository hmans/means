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
Implemented the approved CLI review in PR #20: info; create --title with optional --body-file; editor workflow with hash checks, current-schema validation and recovery copies; delete --hash; show --raw with hidden read compatibility; shared list/search metadata filters; grouped command help; side-effect-free informational commands. Existing structured document inputs remain supported.

Final help design: one canonical description and examples per command, shared by help <command>, <command> --help, and JSON. Removed the separate agent guide and parallel human-help maps to prevent drift. Setup guidance lives in init, integration guidance in mcp, and detailed workflows/implementation limits in docs/guide.md. Bare means points to help and help mcp. Old humans/agents routes are hidden aliases; their JSON now uses command/description/usage instead of guide. Release note marks this response change as breaking.

Editor metadata uses full replacement, preserving literal nulls and removing omitted keys. VISUAL/EDITOR accepts an executable path; use wrapper scripts for arguments. Failed edits retain a private temporary file and report its path. Ordinary external-write race limitations still apply.

Validation: mise test -race, mise exec -- go vet ./..., and git diff --check. Tests compare every command help route with its text/JSON presentation and compatibility aliases, and cover config side effects, info, filters, raw output, create/delete alternatives, replacement validation/precision, editor recovery/conflicts, and schema changes while editing. Documentation and Sweet notes reflect the final behavior.