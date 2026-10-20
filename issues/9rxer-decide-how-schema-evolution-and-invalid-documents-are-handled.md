---
data:
  priority: normal
  status: todo
  tags:
    - schema
    - design
  type: task
id: 9rxer
title: Decide how schema evolution and invalid documents are handled
---
Consumers can edit schema.json directly. Reads and lists parse existing documents without validating against the current schema; creation and updates validate resulting frontmatter. Decide how consumers discover and resolve invalid documents after schema changes or direct edits.

Acceptance criteria:
- Document the schema-evolution and repair workflow.
- Define how invalid documents are reported without preventing access to their contents.
- Coordinate the decision with doctor; automatic repair remains deferred.

Migrated from TODO.md.
