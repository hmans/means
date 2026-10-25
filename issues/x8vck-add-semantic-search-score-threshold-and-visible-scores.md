---
data:
  priority: normal
  status: done
  tags:
    - embeddings
    - search
  type: feature
id: x8vck
title: Add semantic search score threshold and visible scores
---
Add inclusive --min-score and MCP min_score, default 0.30, with validation and filtering before limits. Semantic text tables show scores. Preserve explicit zero and allow -1 to include all scores. Calibration on issues: openai top 0.349, next 0.287; schema validation top 0.441; unrelated banana bread recipe top 0.184. Verify default, overrides, boundary values, errors, has_more, CLI display, and MCP propagation.

Outcome: Implemented inclusive min-score threshold with default 0.30 across collection, CLI, and MCP. Explicit zero is retained; -1 includes all scores. Invalid/nonfinite values and use without semantic mode fail. Filtering precedes limits and has_more. Semantic tables show three-decimal scores while filtering and JSON retain full precision. Updated help, guide, project rules, and release note.

Validation: mise test and race tests for collection/CLI/MCP pass. Live openai query shows only two hits above the default (approximately 0.350 and 0.306), with visible scores.
