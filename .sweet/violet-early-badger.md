---
type: feature
---

Go applications can watch collections for external changes with
`collection.Watch`. Notifications include document and configuration changes,
respect nested collection boundaries, and stop when the caller cancels its
context. The means TUI uses the same watcher.

