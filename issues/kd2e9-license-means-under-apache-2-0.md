---
data:
  priority: normal
  status: done
  tags:
    - releases
    - docs
  type: task
id: kd2e9
title: License means under Apache-2.0
---
Apply Apache-2.0 to the project's source code, documentation, examples, and configuration.

Outcome: Added the official Apache License 2.0 text as LICENSE. README declares the project-wide license and copyright attribution to the means contributors. GoReleaser includes LICENSE in every archive. Omitted the administrative release note at user request.

Validation: License text matches the Apache Software Foundation original. GoReleaser configuration, snapshot builds for all six targets, Go tests, and vet passed. Inspected all six archives and verified their LICENSE bytes match the repository file. git diff --check passed.