---
data:
  priority: low
  status: done
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

## Outcome

Sort keys on `data` fields now use the field's schema `enum` order when one is declared. Implicit, no opt-in: the enum is in the schema, so the order is predictable and follows schema changes on the next call.

- The enum is found through `properties` and local `#/...` `$ref` pointers (a node's own keywords win over its `$ref`). Fields described only through `allOf`/`anyOf`/`oneOf`/conditionals sort literally.
- Any JSON enum values work (strings, numbers, ...). Values outside the enum sort after enum values with the usual type order; descending reverses the whole order; missing/null stay last.
- Collections should declare enums lowest to highest (the issues schema already does: low, normal, high, urgent).

Validation: `mise test` passes; new tests cover `$ref`, nested properties, numeric enums, out-of-enum values, descending, and combinators. Checked `list --sort -data.priority` on ./issues.
