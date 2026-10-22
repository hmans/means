---
data:
  priority: normal
  status: done
  tags:
    - cli
  type: feature
id: pkq0y
title: Render terminal tables and add Markdown show command
---
Use Charm libraries for colored ID/title tables in list/search and Markdown-rendered means show <id>. Preserve list/search JSON contracts and provide show --json equivalent to read --json. Keep read as exact stored Markdown. Preserve diagnostics and limit notices; disable styling for pipes and NO_COLOR.

Outcome: Added Lip Gloss v2 tables with compact teal IDs, purple headers and muted rounded borders. Tables contain only ID/title and wrap to terminal width; diagnostics and limit notices follow. Added means show with Glamour v2 body rendering and a literal title/ID header. Show JSON equals read JSON; read still emits exact source. Piped output and NO_COLOR disable styling. JSON and MCP contracts remain unchanged. Updated help, guide, README and project instructions.

Validation: mise test and git diff --check passed. Tests cover table columns, omitted metadata/excerpts, diagnostics, limits, Unicode wrapping, empty results, plain output, rendered Markdown, show/read JSON equivalence, errors and JSON help. Smoke-tested real issues list/show and verified ANSI colors on a PTY with NO_COLOR unset; default environment with NO_COLOR stayed plain.