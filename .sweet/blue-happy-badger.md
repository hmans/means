---
type: feature
---

Sorting by a field whose schema declares an `enum` now follows the enum order instead of alphabetical order. For example, with `enum: [low, normal, high, urgent]`, `--sort -data.priority` lists urgent items first. Values outside the enum sort after the enum values.
