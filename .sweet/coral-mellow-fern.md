---
type: feature
---

Extend means doctor with offline warnings for missing or stale document embeddings, orphaned vectors, and corrupt or incompatible embedding caches. Reports include a means embed repair hint and preserve all files. Embedding findings appear as WARNING in text and in a separate JSON warnings array; warnings alone exit 0. Document validation errors and failures to complete the check still exit 1.

Style text reports with a colored status summary, grouped errors and warnings with counts, and readable paths and messages. Disable colors for pipes and NO_COLOR; preserve JSON output.
