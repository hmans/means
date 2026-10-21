---
data:
  priority: normal
  status: done
  tags:
    - storage
    - docs
  type: task
id: 6xyq0
title: Treat nested schemas as collection boundaries
---
A directory containing means.schema.json starts a separate collection. Parent operations must not cross that boundary, even when the child schema is invalid. Ordinary subdirectories remain part of the selected collection.

Acceptance criteria:
- Apply the boundary to list, search, doctor, read, update, and ID allocation.
- Keep IDs, schemas, and workflow guidance local to each collection.
- Test nested collections and invalid schema boundaries, and document changes in ownership when schemas are added or removed.

Outcome:
- The shared document scan stops at any subdirectory containing a means.schema.json entry without loading or following it. Inspection failures stop the scan instead of exposing child documents.
- Added tests for ordinary subdirectories, independently valid parent/child data, local guidance, reused IDs and allocation, isolated updates, invalid JSON, directory and dangling-symlink markers, and boundary addition/removal.
- Updated CLI help, MCP guidance, the usage guide, and project instructions.

Validation: mise test passed for all packages; git diff --check passed; doctor reported 20 healthy issue documents with no diagnostics.
