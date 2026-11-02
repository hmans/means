---
type: fix
---

CLI JSON errors now remain structured after malformed flag values, and unknown
schema or instructions subcommands fail instead of reporting success. Array
filters now compare complete values, including objects nested inside arrays,
while preserving numeric precision.
