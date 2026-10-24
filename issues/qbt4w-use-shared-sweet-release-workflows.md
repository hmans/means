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
Use the published Sweet Release bump and release workflows in Means. Create release PRs on main pushes, gate version tags on Go checks, retain the existing GoReleaser publishing policy, and document token setup and recovery. Verify locally and in PR CI before merge.