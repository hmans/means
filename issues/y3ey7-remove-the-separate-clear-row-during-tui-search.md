---
data:
  priority: normal
  status: done
  tags:
    - tui
  type: bug
id: y3ey7
title: Remove the separate Clear row during TUI search
---
Moved Clear into the top toolbar when search or data filters are active. Search alone no longer reserves a status row. Filter summaries and status messages retain their row and full width. Updated the guide and added a release note. Verified with mise test, including contextual toolbar and search-row regression checks, and git diff --check.