---
data:
  priority: low
  status: todo
  tags:
    - cli
    - mcp
    - search
  type: feature
id: gs8qf
title: Sort string fields by schema enum order
---
Follow-up to h3brd. `--sort data.priority` compares strings literally (high, low, normal, urgent), so enum-like fields such as priority cannot be sorted by their meaning.

Idea: when a sort key addresses a field whose schema declares a string `enum`, order values by enum position (values outside the enum after enum values). Open questions: resolving the field schema through `$ref`/`allOf`/`oneOf`, opt-in vs. implicit behavior (e.g. a key suffix or MEANS.md annotation), and keeping it predictable when the schema changes.

Deferred from h3brd to keep the first sort implementation simple and schema-independent.