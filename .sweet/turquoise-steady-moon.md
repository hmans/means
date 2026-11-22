---
type: feature
breaking: true
---

Documents now record `created_at` and `updated_at`. Means sets both on creation and bumps `updated_at` whenever a document changes, including appends and body edits. Use them in `--where` and views (for example `date(created_at) > now() - duration("168h")`), sort with `--sort -updated_at`, or show them as view columns. Documents created earlier have no `created_at` and gain `updated_at` on their next write. Body-only edits now rewrite the frontmatter, so YAML comments and formatting there are no longer kept. Older means versions cannot read documents that have these fields.
