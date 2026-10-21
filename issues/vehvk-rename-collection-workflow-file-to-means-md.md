---
data:
  priority: normal
  status: done
  tags:
    - mcp
    - docs
  type: task
id: vehvk
title: Rename collection workflow file to MEANS.md
---
Use MEANS.md as the collection-local workflow file instead of instructions.md.

Acceptance criteria:
- Load MEANS.md for collection guidance and exclude the root file from document operations.
- Rename the repository workflow file and update CLI help and documentation, including migration guidance.
- Verify collection and MCP behavior with the Go test suite.

Outcome:
- Renamed the repository workflow to MEANS.md and updated collection loading, MCP descriptions, CLI help, and documentation links.
- Documented the required rename for existing collections; instructions.md is now an ordinary document filename.
- Updated fixtures and added coverage for root-only reservation and absence of legacy guidance fallback.

Validation: mise test passed for all packages; git diff --check passed; the issues collection passed doctor with 19 documents and no diagnostics.
