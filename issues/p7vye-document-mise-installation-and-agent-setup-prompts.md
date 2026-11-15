---
data:
  priority: normal
  status: done
  tags:
    - docs
  type: task
id: p7vye
title: Document mise installation and agent setup prompts
---
Add a released-binary mise installation command and short, copyable agent prompts to README.md. Keep source-build instructions and the README concise.

Outcome: Added global and project-local mise installation guidance and a shell setup link. Added two simple prompts for project memory and project planning, using the wording requested by the user. Preserved Homebrew and source-build instructions.

Validation: Checked installation syntax against official mise GitHub backend documentation. mise ls-remote github:hmans/means resolved published versions. Reviewed the final prompts and ran git diff --check. Did not change the machine installation or run Go tests for this documentation-only change.