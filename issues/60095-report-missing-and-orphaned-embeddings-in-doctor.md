---
data:
  priority: normal
  status: done
  tags:
    - embeddings
    - diagnostics
  type: feature
id: "60095"
title: Report missing and orphaned embeddings in doctor
---
Extend doctor with offline read-only checks for missing/stale document vectors and orphaned vectors. Reuse diagnostics and unhealthy exit behavior. Disabled collections skip embedding checks. Report corrupt/incompatible cache without changing it. Verify no API requests or cache writes and document behavior.

Outcome: Doctor now reads the existing cache without locking, creating files, or calling an API. It reports missing/stale vectors on document paths, sorted orphan diagnostics on the cache path, and corrupt/incompatible cache diagnostics. Embedding problems set healthy=false and retain the existing stdout report/exit 1 contract. Disabled collections skip checks. Schema-invalid documents retain ownership; duplicates and nested collections do not. Updated CLI help and guide, with a consolidated feature note.

Validation: mise test passes, including CLI text/JSON output, missing/corrupt/disabled cache behavior, unchanged cache bytes, no API calls, direct edits/deletes, nested boundaries, duplicate IDs, and schema-invalid ownership.


Behavior correction: Embedding findings are now warnings, not unhealthy document diagnostics. Text labels them WARNING; JSON includes a separate warnings array when present. Warnings do not change healthy or cause exit 1. Document validation errors and failures to read/check the collection still fail. Updated tests cover warning-only success and mixed document errors plus warnings; updated help, guide, and the existing release note.


Presentation follow-up: Doctor text output now has a colored status summary, grouped error/warning counts, and separate indented paths and messages. Reuses terminal rendering so pipes and NO_COLOR strip color; JSON remains unchanged. Stored values are sanitized as plain terminal text. mise test passes and the issues report was checked interactively.
