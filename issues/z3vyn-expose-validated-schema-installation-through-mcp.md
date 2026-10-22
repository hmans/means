---
data:
  priority: normal
  status: done
  tags:
    - schema
    - mcp
  type: feature
id: z3vyn
title: Expose validated schema installation through MCP
---
Add schema_set backed by collection.SetSchema, with an explicit optional force boolean. Preserve validation diagnostics on rejection and forced installation, JSON number precision, and recovery after an external invalid-schema edit. Test both supported protocol versions and structured-only responses.

Outcome: Added the destructive schema_set tool with object/boolean schema input and optional force (default false). Uses collection.SetSchema without loading the old schema first, allowing a running server to repair external schema corruption. Compatibility rejection returns isError plus structured diagnostics and explanatory text in every output mode. Forced installation retains diagnostics. Updated MCP help, workflow instructions, and usage documentation.

Validation: mise test passed. Protocol tests cover both supported versions and both output modes, invalid inputs and schemas, rejection preservation, forced installation, invalid-schema recovery, exact large integers and decimals, refreshed info, and startup metadata remaining a snapshot. git diff --check passed; doctor reported 22 healthy issue documents.
