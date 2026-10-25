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
Use Sweet's shared bump and tag workflows. Means owns its tag-triggered GitHub publication workflow and GoReleaser configuration.

The first integration (PR #6) published Means 0.1.1 end to end. RELEASE_PR_TOKEN is configured. The previous shared-publication design is superseded by the narrower scope in chattocorp/sweet-release#47.

Implementation: bump creates release PRs; main CI runs race tests and vet before tag.yml pushes committed root-package v* tags. Both shared workflows use RELEASE_PR_TOKEN and build Sweet from the pinned workflow revision. Means' Release workflow extracts changelog notes, builds six archives, verifies the binary version and checksums, and publishes through gh. Retries retain tags, recreate incomplete drafts, and skip publication for published releases. Prereleases are marked explicitly.

Validation: workflow lint, Go tests/vet, all six snapshot archives, native binary version, checksums, and changelog extraction. Hosted PR CI verifies the consumer configuration.

Sweet #47 is merged. Both workflow calls now pin its merged revision 01911da0f43f6147d1918c5b4e776abcdd9b7c3a, whose tree matches the reviewed Sweet commit.

Remaining: merge Means PR #16 and confirm the next live release end to end.