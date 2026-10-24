---
data:
  priority: normal
  status: done
  tags:
    - releases
  type: task
id: 1q88b
title: Set up releases with Sweet Release
---
Configure Sweet Release 0.2.0 for the root Go package. Add a first-release note, reproducible release commands, GitHub binary publishing, and contributor documentation. Verify release preparation and builds locally without publishing a release.

Outcome: Added root Sweet configuration and a 0.0.0 manifest with an initial feature note for 0.1.0. Added pinned mise tasks, GoReleaser archives for Linux/macOS/Windows on amd64/arm64, and a tag-triggered GitHub workflow that validates the manifest and changelog before uploading a draft release. Documented change notes, release preparation, tagging, draft publication, and retry steps in docs/releases.md.

Validation: mise test and go vet passed. GoReleaser configuration and actionlint with ShellCheck passed. The snapshot built all six archives; SHA-256 checksums and the native binary version passed. A temporary Git repository verified bumping to 0.1.0, extracting changelog notes, validating the tag/version, and creating the annotated tag. No project tags or GitHub releases were created; remote publishing awaits merge and a release tag.