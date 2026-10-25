---
data:
  priority: normal
  status: in_progress
  tags:
    - releases
  type: task
id: d6ztp
title: Simplify consumption of Sweet GitHub workflows
---
Use one shared release call for PR previews and main releases. Let the workflow derive the commit range and use its tested CLI default. Keep the GoReleaser preset, local build configuration, and existing permissions/concurrency rules. Verify the consumer against the new shared planner, tag stage, builder asset contract, and summaries. Keep the PR draft until Sweet is merged and an immutable workflow version is published; replace the temporary test SHA before merging Means.