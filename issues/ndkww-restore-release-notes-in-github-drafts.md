---
data:
  priority: normal
  status: done
  tags:
    - releases
  type: bug
id: ndkww
title: Restore release notes in GitHub drafts
---
The 0.1.0 draft had all binary archives but blank release notes. GoReleaser changelog.disable skipped loading the supplied Sweet Release notes.

Outcome: Removed that setting and added a comment explaining why changelog processing must stay enabled. Added a Sweet fix note. Restored the existing GitHub draft notes from Sweet output without publishing or replacing assets.

Validation: GoReleaser configuration check passed. A local non-snapshot release with publish, announce, validate, and before hooks skipped completed successfully and loaded the supplied notes. GitHub notes match Sweet output exactly; the release remains a draft with all seven assets. git diff --check passed.