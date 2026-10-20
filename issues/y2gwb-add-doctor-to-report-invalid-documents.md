---
data:
  priority: normal
  status: done
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

Outcome: Added `means doctor` with structured JSON and text reports for malformed documents, invalid means envelopes, duplicate IDs, and application schema violations. Reports include relative paths, stored IDs when readable, and validation errors. Healthy collections exit 0; unhealthy collections write the full report to stdout and exit 1. Collection-level failures use stderr. Doctor only reads files. Updated README and CLI help.

Validation: `GOCACHE=/tmp/means-go-cache mise test` passed. `mise means doctor --path ./issues --json` reported 13 checked documents and a healthy collection. `git diff --check` passed.
