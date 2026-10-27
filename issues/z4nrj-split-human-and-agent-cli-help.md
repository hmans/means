---
data:
  priority: normal
  status: done
  tags:
    - cli
    - docs
  type: feature
id: z4nrj
title: Split human and agent CLI help
---
Implemented concise, identical help, help humans, and --help output. Bare means prints only a banner and available commands, with structured command entries in JSON mode. Detailed CLI guidance is in the embedded help agents guide; means agents remains a hidden compatibility alias. Updated init discovery hints, README, usage guide, and release note. Command-specific and nested command help remain available.

Validation: mise test and git diff --check pass. Tests cover audience routing, text/JSON parity, banner scope, compatibility alias, nested command help, and help availability with an invalid user config. Manually inspected help and bare-command output.