---
data:
  priority: normal
  status: done
  tags:
    - cli
  type: bug
id: rs3rn
title: Improve show reading width and bullet styling
---
User screenshot showed overly long prose and plain bullet markers in means show taarx. Reproduced full-terminal-width wrapping in the installed binary at 40/80/160 columns; could not reproduce a complete absence of wrapping. Glamour default dark style inherits gray bullet color.

Changed show to use an at-most-80-column reading width, shrinking for narrow terminals. Styled bullet Prefix (rather than inherited BlockPrefix) with a teal accent, leaving body text neutral. Updated help and guide. JSON/read/plain-output contracts remain unchanged.

Validation: mise test and git diff --check passed; regression tests assert paragraph/nested-list width and intact content at 24/40/80/160 columns plus explicit bullet ANSI color. Checked real taarx output on a PTY. Rebuilt ~/.local/bin/means with mise deploy.