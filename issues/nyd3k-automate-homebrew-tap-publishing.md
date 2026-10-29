---
data:
  priority: normal
  status: in_progress
  tags:
    - releases
  type: task
id: nyd3k
title: Automate Homebrew tap publishing
---
Generate a Homebrew cask with release archives and publish it to hmans/homebrew-tap after stable GitHub releases. Support retries without rebuilding cask checksums or downgrading the tap. Document the scoped Actions token and installation.

Implemented: GoReleaser generates means.rb for all four macOS/Linux targets; Release uploads it with the archives. A serialized Homebrew job downloads the published file, skips prereleases, avoids duplicate commits, and refuses downgrades. Release documentation, the install command, and a Sweet feature note are included.

Validation: GoReleaser config, all six snapshot builds, Go tests/vet, Ruby syntax, all four cask checksums, and actionlint with ShellCheck passed. Disposable local tap repositories verified initial publication, upgrade, unchanged retry, and older-release retry.

The user chose a new, separate hmans/homebrew-tap. Created the public repository with an initial README and verified its canonical name and main default branch. Beans remains in hmans/homebrew-beans. No Means release has been published.

The user installed HOMEBREW_TAP_TOKEN; its presence in hmans/means was verified. Secret values and permissions cannot be inspected through the secret listing. Remaining: Merge the implementation and verify a live stable release; then brew install --cask hmans/tap/means becomes available.