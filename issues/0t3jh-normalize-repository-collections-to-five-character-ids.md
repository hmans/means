---
data:
  priority: normal
  status: done
  tags:
    - storage
    - examples
  type: task
id: 0t3jh
title: Normalize repository collections to five-character IDs
---
Regenerate longer document IDs in issues/, example/issues/, and example/memory/ using means. Use canonical ID/title-slug filenames, preserve already canonical five-character IDs, and update any repository references.

Acceptance criteria:
- Every stored document has a unique five-character ID and canonical filename.
- Titles, application data, and bodies are preserved.
- All documents remain readable and validate against their collection schemas.

Outcome: Regenerated 18 longer IDs through means; retained the two existing five-character IDs. All 20 documents across the three collections now have canonical filenames and collection-unique five-character IDs. No external repository references to the old IDs were found. Updated the README migration note.

Validation: Compared every document before and after migration; titles, application data, and bodies were preserved. Read every document by its new ID and validated each through a no-op hash-checked update, preserving exact hashes.
