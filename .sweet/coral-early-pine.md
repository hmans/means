---
type: fix
---

Creating or updating a document no longer silently changes frontmatter values that YAML reads back differently. Strings containing the Unicode line breaks NEL, LS, or PS (including titles), DEL and C1 control characters, or a leading tab are now stored exactly. Numbers outside the float64 range, such as `1e400`, were previously stored as strings; they are now rejected with an error that names the field, and nothing is written.
