---
data:
  priority: normal
  status: done
  tags:
    - docs
    - examples
  type: task
id: 3j4kp
title: Remove obsolete example collections
---
The repository's own issues collection now exercises means through CLI and MCP, so the separate fictional example directory is redundant.

Acceptance criteria:
- Remove example/ and its nested Codex configuration and sample collections.
- Remove tasks and live documentation/agent guidance that point to example/ while preserving historical issue outcomes.
- Keep useful generic initialization and integration examples independent of a checked-in sample collection.
- Run tests and doctor on the real issues collection.

Outcome: Removed the entire example/ tree, including fictional issues, memory, and nested Codex configuration. Removed its mise tasks and current references from README, AGENTS.md, root MCP config, CLI MCP help, and issue workflow instructions. README examples now use the real issues collection. Kept completed issue records and TODO milestones as history.

Validation: GOCACHE=/tmp/means-go-cache mise test passed. GOCACHE=/tmp/means-go-cache mise means doctor --path ./issues --json reported a healthy collection with 16 documents. A reference scan found example/ only in historical issue records and this removal issue. git diff --check passed.
