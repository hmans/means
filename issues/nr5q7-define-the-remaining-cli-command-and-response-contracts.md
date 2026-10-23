---
data:
  priority: normal
  status: done
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

Outcome:
- Audited deletion, search/filtering, and doctor against their CLI implementations, collection methods, existing tests, and usage guide. Deletion already documents its input, outputs, errors, and concurrency limitations.
- Clarified empty search queries, result fields, no-match success, and negative limits in help and docs; added a literal-search example.
- Defined doctor input constraints, inspected-file counts, diagnostic fields, and report-versus-error output behavior.
- Clarified the schema-set rejection report in root help. Long descriptions and examples remain available through human-readable and JSON help.
- Confirmed collection operations remain independent of terminal presentation. MCP search and deletion call the same collection methods; doctor is reusable collection logic without an MCP tool. No runtime behavior changes were required.

Validation:
- mise test passed across CLI, collection, and MCP packages.
- Inspected generated search/delete JSON help and doctor text help.
- git diff --check passed.
