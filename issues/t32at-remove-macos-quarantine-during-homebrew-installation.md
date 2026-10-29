---
data:
  priority: high
  status: done
  tags:
    - releases
  type: bug
id: t32at
title: Remove macOS quarantine during Homebrew installation
---
Added the same macOS-only postflight xattr hook used by hmans/homebrew-beans to the means GoReleaser cask configuration. It removes com.apple.quarantine from the staged means binary. Updated release documentation and added a Sweet fix note. Validation: mise run release:check passed with GoReleaser 2.18.2. The change is local and has not yet been published to the tap.