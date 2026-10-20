---
data:
  priority: normal
  status: todo
  tags:
    - sdk
    - design
  type: task
id: m12zm
title: Use integrations to decide what SDK support is needed
---
Software consumers are a priority, but CLI and MCP may cover their needs. Use actual integrations, especially ChattoBot, to decide whether an SDK adds value.

Acceptance criteria:
- Record integration needs that CLI or MCP do not adequately cover.
- Decide whether to provide an SDK, its language and scope, or defer it.
- Preserve ordinary CLI commands as short-lived processes.

This issue requests an evidence-based decision, not an SDK implementation commitment.

Migrated from TODO.md.
