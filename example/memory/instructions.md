# Plain memory

Use this collection to remember and recall durable facts, preferences, context,
and notes about the fictional example application. The seeded memories are
illustrative; they are not facts about the user or the means project.
Use the separate `issues` collection for actionable work and its progress.

## Remembering

Call `info` for the current schema. Supply a descriptive `title` and optional
`tags`; write the memory itself in the Markdown body. No status, priority, or
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

Search and update tools are not implemented yet. Filter list results yourself;
inspect per-document errors. For a requested correction, read the memory first.
If filesystem access is available, edit its existing file under the path returned
by `info`, preserving its ID and schema-compatible frontmatter, then read it back.
Reads verify parsing, not schema compliance. If filesystem access is unavailable,
explain the limitation rather than creating a contradictory replacement memory.
