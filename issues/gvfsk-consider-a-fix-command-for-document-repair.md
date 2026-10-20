---
data:
  priority: low
  status: todo
  tags:
    - deferred
    - diagnostics
  type: task
id: gvfsk
title: Consider a fix command for document repair
---
Later / under consideration: a fix command could rewrite documents that doctor identifies as invalid. Automatic document repair is deferred.

Before implementation:
- Use real invalid-document cases to establish which repairs are useful and unambiguous.
- Decide preview, confirmation, and stale-content behavior.
- Specify what cannot be repaired automatically and how it is reported.

Do not begin implementation unless the user brings this into scope.

Migrated from TODO.md.
