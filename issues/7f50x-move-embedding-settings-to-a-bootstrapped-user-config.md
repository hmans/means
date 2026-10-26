---
data:
  priority: normal
  status: done
  tags:
    - embeddings
    - configuration
  type: feature
id: 7f50x
title: Move embedding settings to a bootstrapped user config
---
Keep collection configuration portable and machine setup local. Bootstrap a commented, disabled ~/.config/means/config.yaml (respect XDG_CONFIG_HOME) on CLI invocation/MCP startup without overwriting existing files. Users edit it manually; no configuration command. Embedding settings apply to every collection on the machine. Credentials remain environment variables. Retain per-collection disposable caches and explicit embed refresh. Remove collection embedding settings with actionable migration errors. Bootstrap write failures must not break offline commands; preserve structured CLI errors and MCP stdout. Cover config parsing/bootstrap, CLI/MCP automatic maintenance, disabled behavior, migration, and isolated tests; update help/docs and release note.

## Outcome

Implemented machine-local configuration at `$XDG_CONFIG_HOME/means/config.yaml`,
defaulting to `~/.config/means/config.yaml`. CLI invocation (including help/version)
and MCP startup create a commented, disabled example when missing, publishing
the complete file without overwriting existing files during concurrent startup.
Users edit it manually. Bootstrap failures warn and permit offline use; existing
invalid/unreadable configs fail with a path-specific error. CLI help/version
remain accessible. Credentials and endpoint overrides stay in the environment.

CLI reads host settings per invocation; MCP captures them at startup and still
reloads collection schema/workflow per call. Collection code receives settings
explicitly. Existing compatible local float caches retain their profile identity.
Legacy MEANS.md embedding fields are rejected with migration guidance. Removed
the issues collection's embedding settings and updated its workflow guidance,
CLI/MCP help, agent instructions, usage guide, README, and breaking feature note.

Validation: `mise test -race`, `go vet ./...`, and `git diff --check` passed. Tests
cover concurrent/non-destructive bootstrap, strict parsing and provider defaults,
write-failure warnings, JSON framing, host-wide CLI configuration, disabling and
reloading, MCP snapshot/restart behavior, and legacy migration errors. Existing
embedding integration tests use isolated host settings and mock APIs. The built
CLI bootstrapped the actual commented user config successfully; doctor validated
all 42 issues with no diagnostics. No document content was sent to an external
embedding service during this work.
