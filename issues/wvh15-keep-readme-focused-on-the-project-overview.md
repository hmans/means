---
data:
  priority: normal
  status: done
  tags:
    - docs
  type: task
id: wvh15
title: Keep README focused on the project overview
---
The root README is a long command reference. Keep it concise and focused on what means is, its current state, and where to start.

Acceptance criteria:
- Shorten README.md to a readable overview with a small working example and links to detailed guidance.
- Move or remove operational details; keep needed reference information in docs/ or command help.
- Codify the README scope rule in AGENTS.md for future work.
- Check internal links and run relevant validation.

Outcome: Reduced README.md from 512 lines to 45 lines focused on the purpose, current capabilities, a first-use example, and links. Added docs/guide.md for collection format, document operations, MCP, and migration guidance, with CLI help as the detailed command reference. AGENTS.md now requires a concise overview README and moves detailed workflows and contracts to docs/ or CLI help.

Validation: GOCACHE=/tmp/means-go-cache mise test passed. The README init/create/list example worked in a temporary collection. All local links in README.md, docs/guide.md, and AGENTS.md resolve. git diff --check passed.
