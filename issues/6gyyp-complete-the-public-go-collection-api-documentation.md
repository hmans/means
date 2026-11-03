---
data:
  priority: normal
  status: done
  tags:
    - sdk
    - docs
  type: task
id: 6gyyp
title: Complete the public Go collection API documentation
---
Follow up on m12zm and PR #44. Consumers should be able to use the public collection package through Go documentation and editor tooltips.

Acceptance criteria:
- Document exported types, fields, defaults, result semantics, and errors.
- Describe datetime operators, search constraints, embedding environment settings, and instruction replacement/reload behavior.
- Add executable examples for search, schema rejection, conflict handling, and workflow updates.
- Verify comments against implementation, inspect go doc output, and run the examples.
- Preserve runtime behavior and API signatures.

Outcome:
- Every exported declaration and field now has code documentation. Added search mode/default/filter rules, datetime operators and timestamp limits, result/body/hash semantics, pagination, error contracts, embedding environment names and defaults, and configuration replacement/reload behavior.
- Added executable examples for filtered full-text search with datetime conditions, schema rejection with diagnostics and reload, stale-write conflict reconciliation, instruction replacement and clearing, and embedding defaults. Seven examples now run without network access.
- Expanded the package overview and constructor guidance so consumers can work from Go documentation.

Validation:
- mise test -run '^Example' -count=1 passes all seven examples.
- mise test ./collection (the task also includes ./...) and go vet ./collection pass.
- Inspected rendered go doc -all output. A one-off AST audit found no undocumented exported declarations or fields.
- A Go-token comparison across all 18 production collection files confirms that only comments/formatting changed; runtime code and API signatures are unchanged.
- Full PR self-review found no remaining significant issues.
