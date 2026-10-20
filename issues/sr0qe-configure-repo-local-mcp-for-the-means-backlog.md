---
data:
  priority: normal
  status: done
  tags:
    - mcp
    - integration
  type: task
id: sr0qe
title: Configure repo-local MCP for the means backlog
---
Configure Codex at the repository root to launch means against the actual issues/ collection. Add a mise task bound to the repository root, document MCP usage and CLI fallback, and preserve the example's fictional issues/memory configuration.

Acceptance criteria:
- Root issues registration resolves to the real project backlog.
- Server starts through its configured command and exposes info/list/read/create/update.
- Nested example configuration continues to select the fictional collections.
- Startup, collection identity, and read operations are verified over MCP.

Outcome: Added root .codex/config.toml issues registration and the root-bound issues-mcp mise task. Updated agent and README guidance. The nearer example configuration keeps its fictional issues registration.

Validation: codex mcp get issues --json resolves issues-mcp at the repository root and example-mcp under example/issues. The actual configured commands passed MCP initialization, five-tool discovery, info, list, and read from root, issues/, and example/issues/. Marked this issue done through the new root MCP connection.
