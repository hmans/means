---
data:
  priority: normal
  status: done
  tags:
    - cli
    - mcp
  type: feature
id: 34f40
title: Define and implement document deletion
---
Deletion completes the basic CRUD surface. It requires the stored document ID and current SHA-256 content hash. Missing or duplicate IDs, malformed envelopes, and stale or invalid hashes fail. Only the document file is permanently removed; directories, configuration, and unrelated files remain. Existing application data need not satisfy the current schema. There is no trash, force option, confirmation prompt, or successful replay after removal.

Acceptance criteria:
- Decide whether deletion requires a content hash and how missing documents are handled.
- Implement the agreed behavior in the shared collection layer, CLI, and MCP.
- Document responses and errors, and test that unrelated files and collection configuration are preserved.

Outcome:
- Added collection.Delete, CLI delete with structured --input, and the destructive MCP delete tool. Results contain only the deleted ID.
- Documented permanent removal and the optimistic concurrency limitation: a concurrent write can race the final removal, as with updates.
- Added collection tests for moved documents, reused child IDs, schema evolution, preserved files/directories/configuration, stale/invalid hashes, invalid/missing IDs, duplicates, malformed envelopes, hidden paths, symlinks, nested boundaries, and reserved workflow files.
- Added CLI tests for file/stdin input, text/JSON output, invalid input, error streams, replay, and human/JSON help. Extended MCP stdio tests for both supported protocol versions, destructive annotations, invalid arguments, stale hashes, deletion after schema changes, and repeated calls.

Validation: mise test passed for all packages; git diff --check passed; doctor reported 21 healthy issue documents.

Migrated from TODO.md.
