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
Use shared Sweet Release bump and release workflows with a thin consumer configuration. Keep all GitHub lifecycle behavior in the reusable workflows and only binary build configuration in Means.

The first integration (PR #6) successfully published Means 0.1.1 end to end, including automatic PR checks, tags, and publication. The maintainer configured RELEASE_PR_TOKEN.

Follow-up: remove Means tag-triggered publishing workflow and call the shared GitHub lifecycle with the GoReleaser preset. The shared workflow prepares notes/drafts, builds, verifies/uploads assets, and publishes; retries skip published releases. Means retains .goreleaser.yml, uses GITHUB_TOKEN for tagging/publication, and needs RELEASE_PR_TOKEN only for bump PRs.

Validation: local six-platform snapshot build and native binary version pass; the shared uploader validates all seven assets. Configuration and workflow lint pass. Companion Sweet and Means PR CI and a future live release will verify the simplified path.