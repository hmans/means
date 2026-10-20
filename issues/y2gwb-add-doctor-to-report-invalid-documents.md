---
data:
  priority: normal
  status: in_progress
  tags:
    - schema
    - diagnostics
  type: feature
id: y2gwb
title: Add doctor to report invalid documents
---
Add a diagnostic command that identifies malformed Markdown/frontmatter and documents that do not satisfy the current collection schema.

Acceptance criteria:
- Report affected document IDs and actionable validation errors.
- Define structured output and exit status for healthy and unhealthy collections.
- Inspect the collection without rewriting documents.
- Test healthy, malformed, and schema-invalid documents.

Migrated from TODO.md.
