---
data:
  priority: normal
  status: in_progress
  tags:
    - releases
  type: task
id: qbt4w
title: Use shared Sweet Release workflows
---
Use the published Sweet Release bump and release workflows in Means. Create release PRs on main pushes, gate version tags on Go checks, and publish GitHub Releases automatically after all GoReleaser uploads succeed. Document token setup and recovery.

Implemented in PR #6: pinned reusable workflows and Sweet 0.3.0, Go race tests and vet, read-only PR release planning, and automatic publication after draft uploads. Local race tests, vet, workflow lint, GoReleaser configuration validation, and the initial consumer PR CI passed. The maintainer will add RELEASE_PR_TOKEN before merge. Live bump/tag/publication verification remains pending merge and the next release PR.