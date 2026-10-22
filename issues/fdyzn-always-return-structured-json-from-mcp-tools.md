---
data:
  priority: normal
  status: done
  tags:
    - mcp
    - cli
  type: task
id: fdyzn
title: Always return structured JSON from MCP tools
---
Make successful MCP results structuredContent-only regardless of --json. Preserve error reporting and exact JSON numbers on the wire. Remove the server output-mode option and the registration flag; --json only formats CLI help and startup errors. Update help, guide and protocol tests.

Outcome: Removed the structuredOnly server option. Successful tools always return structuredContent with empty content; schema rejections retain structured diagnostics and error text, and other failures retain MCP error explanations. --json now only controls CLI help and startup errors. Removed it from the issues registration and updated help and guide.

Validation: mise test and git diff --check passed. Stdio tests cover default output and --json, both supported protocol versions, successful tools, forced and rejected schema changes, error explanations, and exact numeric values on the wire and in storage. The SDK client itself rounds structured numbers through float64; tests distinguish client decoding from server wire precision.
