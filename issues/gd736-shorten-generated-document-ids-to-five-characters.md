---
data:
  priority: normal
  status: done
  tags:
    - storage
  type: task
id: gd736
title: Shorten generated document IDs to five characters
---
Use five-character generated IDs with the existing 32-symbol lowercase alphabet and crypto/rand generator. Keep collection-wide collision detection and allocation retries; preserve existing stored IDs. No NanoID dependency is needed for this fixed power-of-two alphabet.

Acceptance criteria:
- Newly created documents receive five-character IDs through CLI and MCP.
- Existing IDs remain readable and updateable.
- Help and documentation describe the new length.
- Existing collision and concurrent allocation checks continue to pass.

Outcome: Generated IDs now contain five lowercase letters/digits from the existing 32-symbol alphabet. Updated CLI/MCP help and documentation. Existing IDs and collision allocation behavior are unchanged.

Validation: mise test passes, including CLI/MCP workflows and concurrent allocation tests. Created this issue as gd736 and verified a pre-existing 32-character ID still reads and accepts a no-op hash-checked update without changing content.
