---
created_at: "2026-11-22T16:31:40Z"
data:
  priority: normal
  status: done
  tags:
    - schema
    - links
    - doctor
  type: feature
id: agxw9
title: Validate document ID references with an x-means-ref schema keyword
updated_at: "2026-11-22T16:55:14Z"
---
Issue tracking and agent memory need relationships ("parent", "blocked by", "supersedes"). Links should stay application data, not a means primitive: the schema marks which data fields hold document IDs, and means verifies them.

## Design

A custom JSON Schema keyword, `x-means-ref`. JSON Schema cannot add types, and `format` describes syntax, not references. Draft 2020-12 allows unknown keywords and other validators ignore them, so collections stay portable. Register it as a custom vocabulary in `santhosh-tekuri/jsonschema/v6`.

```yaml
$defs:
  docId:
    type: string
    x-means-ref: true
properties:
  parent: { $ref: "#/$defs/docId" }
  blocked_by:
    type: array
    items: { $ref: "#/$defs/docId" }
```

Reuse through `$defs`/`$ref` gives type-like ergonomics.

## Checks

1. Shape (schema validation, every write): value must be a string with a valid means ID shape.
2. Existence (collection-wide): the ID must belong to a document in the same collection. `doctor` reports dangling references as diagnostics. Decide whether create/update also reject unknown IDs (extra scan per write); deletes can still leave dangling refs, so doctor remains authoritative.

## Acceptance criteria

- `x-means-ref: true` compiles in collection schemas, including via `$ref`; invalid keyword values (non-boolean) fail schema compilation.
- Shape violations fail writes like other schema errors.
- Doctor reports dangling references with document, field path, and missing ID.
- Documented in `means help agents` and docs/guide.md.

## Out of scope / later

- `[[id]]` references in Markdown bodies.
- Cross-collection references (the keyword can later accept an object, e.g. a target collection, without breaking `true`).
- Backlinks: expressible as a query once Expr filters (mmed0) land; TUI could make ref fields navigable.


## Outcome

Added the `x-means-ref` keyword as a custom vocabulary (`collection/ref.go`, exported as `collection.RefKeyword`), registered for every collection schema.

- Keyword value must be boolean; `false` is a no-op. Works through `$defs`/`$ref` and array `items`.
- Writes reject marked values that are not document ID strings (same rule as stored IDs).
- `doctor` (and so `schema set`/`schema_set`) reports marked IDs naming no document, with a JSON Pointer under data (`/blocked_by/1`). Only collections whose schema uses the keyword pay for the extra pass.
- Decision: writes do **not** check existence; deletes can leave dangling refs anyway, so doctor is authoritative. Means never rewrites references.

Validation: `mise test` passes; new tests cover keyword compilation, write validation, `$ref` reuse, pointer escaping, and doctor reports. Verified end to end via CLI.
