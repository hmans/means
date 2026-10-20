---
data:
  priority: normal
  status: done
  tags:
    - mcp
    - cli
  type: task
id: gf7tk
title: Reduce MCP command surface friction
---
Improve the command surface based on using the repository issues MCP server.

Acceptance criteria:
- Reduce repeated MCP discovery guidance while retaining collection identity and workflow at startup and current details in info.
- Expose a useful typed update input schema without changing its accepted arguments.
- Avoid duplicating the full JSON result in MCP text content while preserving client compatibility and exact JSON number values.
- Update CLI help and README where behavior changes; validate through MCP protocol tests and mise test.

Each improvement is delivered in its own commit.

Outcome: Shortened startup means guidance and individual tool descriptions while retaining collection identity and workflow at startup; info still returns full current guidance. Replaced update's root anyOf with an equivalent minProperties constraint, preserving valid and invalid input behavior while exposing named arguments to clients. MCP results now use compact JSON. Added opt-in --structured-only to omit successful text copies, and enabled it for this repository's issues MCP registration; the default still includes JSON text for clients that need it. Errors retain explanatory text. Updated README and CLI help.

Validation: GOCACHE=/tmp/means-go-cache mise test passed, including stdio tests for both supported MCP protocol versions and structured-only number precision. JSON CLI help includes --structured-only. git diff --check passed.
