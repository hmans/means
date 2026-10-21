---
data:
  priority: normal
  status: done
  tags:
    - storage
    - docs
  type: task
id: sgeat
title: Use means.schema.json as the collection schema filename
---
Use means.schema.json and MEANS.md as the reserved collection filenames with ordinary filesystem lookup. Update initialization, loading, nested boundaries, help, docs, and repository configuration. Verify that unrelated schema.json files do not establish collection boundaries. No case-insensitive matching or legacy fallback.

Outcome: Renamed the reserved schema and the repository issue schema to means.schema.json. Updated CLI help, MCP guidance, documentation, and fixtures. MEANS.md keeps its name. Ordinary filesystem lookup remains unchanged. Added a regression check that generic schema.json files do not establish boundaries.

Validation: mise test passed for all packages; git diff --check passed; doctor reported 21 healthy issue documents.
