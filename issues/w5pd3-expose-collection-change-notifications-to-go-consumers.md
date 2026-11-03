---
data:
  priority: normal
  status: done
  tags:
    - sdk
    - tui
  type: feature
id: w5pd3
title: Expose collection change notifications to Go consumers
---
Add a context-owned public collection watcher with coalesced invalidations, nested collection boundaries, atomic replacement support, and recovery after missed events. Keep UI state and app discovery in consumers. Make the means TUI use the library API. Verify filesystem changes, cancellation, and recovery; document the contract and limits.

Outcome: Added collection.Watch(ctx, path) and WatchEvent with coalesced invalidations and error reporting. Moved filesystem logic from the TUI into the public package; the TUI uses an adapter and refreshes after errors too. Context cancellation releases resources before closing the event channel. Directory watches recover after overflow and retry reconciliation failures. Documented lifecycle, nested boundaries, snapshot reloads, and notification limits.

Validation: mise test -race, go vet ./..., and git diff --check passed. Tests cover write bursts, atomic replacement, deletion, new directories, nested boundaries, invalid startup paths, cancellation without a reader, root recreation, overflow recovery, and the TUI adapter.