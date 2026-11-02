---
data:
  priority: normal
  status: done
  tags:
    - cli
    - mcp
  type: feature
id: 6m1xp
title: Add offset pagination to list
---
Add list offset support while retaining the existing limit default of 20 and unlimited value 0. Skip valid matching documents in filename order after exact-data and datetime filters. Report collection-wide diagnostics on every page. Expose offset through CLI and MCP, reject negative or invalid values, document pagination, and verify page boundaries, filters, bodies, and has_more.

Outcome: Added CLI --offset and MCP offset, both defaulting to 0, through the shared collection ListWithOptions method. Filters apply before offset and limit; malformed and duplicate documents do not consume either count. Every page retains all diagnostics. Unlimited results start at the offset; an offset at or beyond the end returns an empty array and has_more false. Updated canonical help, the usage guide, and one feature release note.

Validation: mise test -race, mise exec -- go vet ./..., and git diff --check passed. Tests cover filename order, exact-data and datetime filters, bodies, text and JSON output, diagnostics, page boundaries, large offsets, invalid inputs, and MCP pagination with both protocol versions. A CLI smoke check against this issues collection returned the expected page.