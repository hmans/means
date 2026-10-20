---
data:
  priority: normal
  status: done
  tags:
    - storage
    - cli
    - mcp
  type: feature
id: 27a24
title: Establish means-owned document identity and nested application data
---
Implement reserved frontmatter fields id and title, with collection-schema validation limited to data. Generate short collection-scoped IDs and ID/title-based filenames; resolve documents by stored ID regardless of filename. Keep IDs immutable and hash-checked updates lock-free. Update CLI/MCP help, tests, and migrate repository collections without losing content.

Acceptance criteria:
- Create/read/list/update agree on the reserved envelope and data validation.
- IDs are generated, collisions retry, and duplicate stored IDs are reported.
- File renames and moves preserve ID lookup; title updates preserve the existing filename.
- Application fields cannot collide with reserved fields.
- Existing repository documents are migrated with IDs and bodies preserved.
- Tests and vet pass; examples and agent instructions describe the new model.

Outcome:
- Implemented means-owned id/title and schema-validated data across collection storage, CLI, and MCP.
- New documents receive 12-character IDs and ID/title-slug filenames. Stored IDs survive renames and nested moves; title edits retain filenames.
- Candidate allocation uses temporary exclusive files to prevent competing creators allocating the same ID. Updates remain lock-free and hash-checked. Duplicate stored IDs are rejected/reported.
- Migrated 18 repository documents with existing IDs, titles, application fields, and bodies preserved; updated schemas, help, and workflows.

Validation: mise test and go vet ./... pass, including cross-process candidate collisions, duplicate IDs across paths, nested data precision, immutability, stale writes, and MCP workflows. CLI smoke passed. All migrated documents passed schema validation through no-op updates with unchanged hashes.
