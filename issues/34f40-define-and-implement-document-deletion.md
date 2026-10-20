---
data:
  priority: normal
  status: todo
  tags:
    - cli
    - mcp
  type: feature
id: 34f40
title: Define and implement document deletion
---
Deletion is part of the intended CRUD surface but its semantics are still open.

Acceptance criteria:
- Decide whether deletion requires a content hash and how missing documents are handled.
- Implement the agreed behavior in the shared collection layer, CLI, and MCP.
- Document responses and errors, and test that unrelated files and collection configuration are preserved.

Migrated from TODO.md.
