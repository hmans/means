---
data:
  priority: normal
  status: done
  tags:
    - cli
    - mcp
  type: feature
id: 9kwxa
title: Let agents update collection instructions
---
Add CLI and MCP operations to replace the MEANS.md Markdown body. Return a whole-file hash from info and require it for stale-write protection. Preserve exact frontmatter and file permissions, support an empty body, and document the workflow. Verify preservation, conflicts, invalid inputs, and current MCP info after updates.

Outcome:
- Added CLI instructions set with JSON file/stdin input and MCP instructions_set, backed by a shared collection operation. Both take hash and complete replacement instructions. Empty instructions clear the body.
- Info returns a SHA-256 hash of the complete configuration snapshot. Updates preserve exact frontmatter bytes, comments, schema numbers, and permissions, and reject stale or observed concurrent changes. Same-directory replacement retains the existing external-write race limitation.
- MCP info.instructions now returns the exact editable body, with general tool guidance in usage. Per-call info reflects changes; startup metadata remains a snapshot until restart.
- Updated canonical help, embedded agent guide, usage guide, project rules, and collection workflow. Added one feature release note.

Validation: mise test passed for all packages. Tests cover LF/CRLF frontmatter, missing final delimiter newline, exact and empty bodies, permissions, invalid documents, bad hashes and inputs, stale frontmatter/body changes, missing/non-regular files, CLI file/stdin input, and both MCP protocol versions with and without --json. Existing shared replacement tests cover observed concurrent changes and temporary-file cleanup. git diff --check passed. Doctor reported 58 healthy issue documents without diagnostics; local embedding-cache warnings remain.