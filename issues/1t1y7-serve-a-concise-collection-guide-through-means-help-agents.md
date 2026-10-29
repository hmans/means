---
data:
  priority: normal
  status: done
  tags:
    - cli
    - docs
  type: feature
id: 1t1y7
title: Serve a concise collection guide through means help agents
---
Implemented an embedded collection authoring guide at means help agents. Covers collection structure, MEANS.md, schemas, workflows, discovery, and paired configuration/document examples. The guide uses the standard help JSON fields; legacy command-specific agents help remains supported. Added root/init help pointers, README and guide documentation, and an AGENTS.md rule to keep the embedded instructions concise and current. Related history: wgxy8 and mn257.

Validation: mise test passed. Tests cover guide text/JSON consistency, aliases, help without creating collection or user configuration files, existing invalid-config behavior, and validation of the actual embedded examples through doctor. git diff --check passed. Added a Sweet feature release note.