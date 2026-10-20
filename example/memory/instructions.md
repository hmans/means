# Plain memory

Use this collection to remember and recall durable facts, preferences, context,
and notes about the fictional example application. The seeded memories are
illustrative; they are not facts about the user or the means project.
Use the separate `issues` collection for actionable work and its progress.

## Remembering

Call `info` for the current data schema. Supply a descriptive `frontmatter.title`
and optional `frontmatter.data.tags`; write the memory itself in the Markdown
body. Means generates `frontmatter.id`; never supply it on creation or in a patch.
For example: `{"frontmatter":{"title":"Team preferences","data":{"tags":["team"]}},"body":"..."}`. No status, priority, or
fixed memory categories are required.

Before creating a memory, use `list` with `limit: 0` and `read` to check related
entries. Prefer one focused memory per subject. Keep useful context and any
source or date provided by the user. Distinguish confirmed facts from uncertain
observations; do not invent missing details. Store information that is useful
across conversations rather than copying whole transcripts.

Create a memory when asked to remember something or when the task explicitly
calls for maintaining memory. Do not retry a successful `create` call: each
success creates a new document.

## Recalling and correcting

Read relevant memories before answering questions about remembered context.
Inspect the body, not just the title, and mention uncertainty or conflicting
entries when it matters. Report memory IDs when creating or changing records.

Search is not implemented yet. Filter list results yourself and inspect
per-document errors. For a requested correction, call `read` first, then `update`
with the memory's ID, returned `hash`, and only the fields to change. Metadata
objects merge recursively, null removes a field, and arrays replace entirely.
The optional `body` replaces the whole body; retain useful context from the read.

Updates validate the means envelope and data against the collection schema. A stale hash means the content changed:
read again, reconcile your correction with the new content, and retry with its
hash. Do not create a contradictory replacement memory to bypass a conflict.
