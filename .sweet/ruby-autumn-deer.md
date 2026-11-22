---
type: feature
---

Mark schema fields that hold document IDs with `x-means-ref: true` (for example `parent` or `blocked_by` items). Writes reject values that are not document IDs, and `doctor` reports references to documents that do not exist in the collection.
