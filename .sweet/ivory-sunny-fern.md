---
type: feature
---

Add `means tui`, a read-only terminal browser with Markdown previews, live literal text search, schema-driven data field/value filters, refresh, and document diagnostics. Search filters immediately while typing; Enter keeps the search and Esc restores the previous one. Filters use schema titles, descriptions, and enum/const choices, while retaining stored values for existing documents.

Browse with framed panes, padded colored toolbar buttons, selection highlights, and visible scroll ranges. Mouse support includes clicking controls, selecting documents and filter values, focusing panes, and scrolling the pane under the pointer. Compact layouts retain keyboard navigation on smaller terminals.

The toolbar keeps Search, Filter, and Refresh prominent with subdued idle fills and dimmed shortcuts. Clear sits beside active filters, and secondary actions remain clickable in the footer.

Type ahead in both filter steps to find fields and values immediately, with paste, Backspace, and Ctrl+U support. Arrow keys, Enter, and mouse selection work on the narrowed choices.

Array filters offer individual items, including schema item enums, and match documents containing the selected item instead of requiring an exact array combination.

String-array choices show how many documents contain each item, counting each document once and including zero counts for unused schema choices.

Place the preview document ID at the upper right beside the title and remove the Markdown body's outer margin.

Separate preview data from the body with a horizontal rule and highlight its JSON syntax.

Match document list colors to the preview: teal IDs and bright white titles, with a dark selection background.

Display active filters on subtle backgrounds with teal field names, dim operators, and purple values.
