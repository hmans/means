---
data:
  priority: normal
  status: in_progress
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
