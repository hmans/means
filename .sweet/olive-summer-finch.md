---
type: feature
---

Append text to document bodies with `means append`, using inline text, a file,
stdin, or JSON input. Means handles append locking and conflict checks internally
and preserves frontmatter formatting. Supply an optional content hash to require
a specific version. Go applications can use `Collection.Append` with the same
checks.
