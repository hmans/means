---
data:
  priority: normal
  status: done
  tags:
    - cli
    - docs
  type: feature
id: wgxy8
title: Ship agent onboarding guidance through means agents
---
Added means agents with an embedded Markdown guide covering the collection model, schema/workflow design, discovery through project instructions or skills, optional MCP setup, and safe routine operations. Works without a collection and makes no filesystem changes. --json returns the identical guide in a guide field. Root/init help and successful init output point to it; init JSON adds next_steps. Updated README and usage guide.

Validation: mise test and git diff --check passed. Tests verify text/JSON equivalence, operation without a collection, no files created, help discovery and both init output modes. Smoke-tested JSON guide output and installed the updated local binary with mise deploy.

Concision follow-up: Reduced embedded guide from 1,222 to 631 words (48% fewer). Put the schema/workflow/discovery completion rule at the top. Retained a concrete project-instruction example and optional MCP route; moved detailed contracts to command help.

Checks: mise test and git diff --check pass. A temporary collection exercised supplied-schema init, discovery hint, representative valid creation, rejection of invalid data, read/hash/update and doctor using the installed binary. Updated local binary. These checks establish command correctness, not agent compliance.

Proposed behavioral eval: give fresh isolated agents the same task (initialize a collection for project issues; start with means --help), vary only old versus concise agents guide, and inspect artifacts. Grade schema fit, MEANS.md workflow, actual discoverability through project instructions/skill/MCP, representative validation, preservation of existing instructions, and no stray test data. Critical pass requires all of schema, workflow and discovery. Use multiple trials and a second purpose such as research notes; do not infer reliability from one successful run. Independent trials await explicit delegation authorization.
