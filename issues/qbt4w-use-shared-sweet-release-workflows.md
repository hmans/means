---
data:
  priority: normal
  status: done
  tags:
    - releases
  type: task
id: qbt4w
title: Use shared Sweet Release workflows
---
Use shared Sweet Release bump and release workflows with a thin consumer configuration. Keep GitHub lifecycle behavior in the reusable workflows and binary build configuration in Means.

PR #6 first published 0.1.1 through the tag-triggered integration. PRs #8 and #9 removed that local publisher, enabled the shared GitHub lifecycle, and pinned its merged revision. RELEASE_PR_TOKEN is configured for bump PRs; tagging and publication use GITHUB_TOKEN.

Verified live on 2026-10-25: release PR #11 published Means 0.2.0 entirely through the shared workflow (CI run 40175280724). All six archives, checksums, release notes, correct tag target, and Latest were verified; the downloaded macOS binary passed checksum verification and reported 0.2.0. Further caller-default and builder-interface simplification is separate follow-up work.