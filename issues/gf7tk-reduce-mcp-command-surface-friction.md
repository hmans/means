---
data:
  priority: normal
  status: in_progress
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
