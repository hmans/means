---
data:
  priority: normal
  status: todo
  tags:
    - cli
    - design
  type: task
id: nr5q7
title: Define the remaining CLI command and response contracts
---
Initialization, creation, reading, listing, and hash-checked patch updates are implemented. Define the remaining command surface as deletion, search/filtering, and doctor are designed.

Acceptance criteria:
- Document arguments, structured inputs, text/JSON outputs, and failure behavior for each new command.
- Keep help sufficient for an agent to discover and use each workflow.
- Keep collection behavior reusable by MCP and future SDK consumers.

Migrated from TODO.md.
