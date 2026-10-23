package cli

import (
	"fmt"
	"strings"

	"github.com/hmans/means/internal/collection"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

func newSearchCommand(opts *options) *cobra.Command {
	var limit int
	var filters []string
	var literal bool
	var withBody bool
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Find documents by ranked full text and application data",
		Long: `Search visible documents in the collection selected by --path.

The optional query matches case-insensitive Unicode tokens in the title and body.
All terms must match, in any order, possibly across both fields. There is no
stemming, stop-word removal, or query-language syntax. Punctuation separates
tokens; quotes are not operators. A nonempty query with no tokens matches nothing.
Results use relevance ranking with a title boost and ID order for score ties.
Use --literal for the original case-insensitive substring search in filename order.
Omit the query or pass an empty string to filter by data alone, also in filename
order. With no query or filters, all valid documents are eligible. A
text match includes its field (title or body) and a short excerpt. Full-text
excerpts prefer a passage with more distinct query terms; ties prefer the title,
then the earliest passage. A passage is at most 180 characters plus ellipses.
Terms split across distant passages or fields may not all appear in one excerpt. Results
include frontmatter but omit bodies and hashes by default. --with-body requires
--json and adds the Markdown body to each match, including empty bodies as "".
Hashes remain omitted; read by ID before updating a document.

Repeat --filter KEY=VALUE to match fields directly under frontmatter.data.
All filters combine with AND. Plain values such as priority=normal are strings;
JSON values such as count=2, active=true, tags=["search"], or nested={"a":1}
retain their types. Quote the entire KEY=VALUE argument in your shell when it
contains JSON quotes or spaces. Nested objects match recursively by supplied
keys; arrays must match in full. A missing field never matches, even for null.
Duplicate filter keys are rejected.

--limit defaults to 20 matching documents; 0 returns all; negative values fail.
has_more reports whether further matches exist. Malformed documents and duplicate IDs are
omitted from matches and reported separately in diagnostics, regardless of
query or result limit. Each diagnostic has an ID when readable, relative path,
and error. Application data that fails the current schema remains searchable.
Hidden paths, symlinks, and root MEANS.md are skipped.
Subdirectories containing means.schema.json are separate collections and are skipped,
even if their schema is invalid.

--json returns {"documents":[...],"diagnostics":[],"has_more":false}.
Each document has id and frontmatter. Text matches also have a match object,
for example {"field":"title","excerpt":"..."}. field is "title" or "body";
data-only results omit match.
No matches returns an empty documents array and exits 0; inspect diagnostics too.
Text output is a colored ID/title table, followed by diagnostics and limit notices.
Pipes and NO_COLOR disable styling. Metadata and excerpts are available with --json.
Search exits 0 even with per-document diagnostics. Invalid arguments or
collection-level errors exit 1 and write stderr, as {"error":"..."} in JSON
mode. Full-text search refreshes a disposable Bleve index in .means/search/ before
each query. Every visible document is read and hashed to catch filesystem edits;
only changed documents are reindexed. Parsed frontmatter and body offsets are
cached in .means/documents.jsonl; unchanged files skip YAML parsing. Missing,
corrupt, or obsolete metadata caches are rebuilt. A lock in .means/search.lock serializes
searches with a 5-second wait. No daemon or watcher is used; MCP also closes the
index after each call. Missing, obsolete, or unopenable indexes are rebuilt.
Symlinked cache paths and permission errors fail. Remove .means/search/ while
no search is running to force a rebuild. Results reflect the files observed
during the scan, not an atomic filesystem snapshot. Literal and data-only searches
do not use or write an index. Source documents are never changed.`,
		Example: `  means search schema --path ./issues --json
  means search "deployment timeout" --literal --path ./memory --json
  means search --path ./issues --filter priority=normal --filter type=feature --json
  means search memory --filter 'tags=["important"]' --limit 0 --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if withBody && !opts.json {
				return fmt.Errorf("--with-body requires --json")
			}
			filter, err := parseSearchFilters(filters)
			if err != nil {
				return err
			}
			query := ""
			if len(args) == 1 {
				query = args[0]
			}
			c, err := collection.Load(opts.path)
			if err != nil {
				return err
			}
			result, err := c.Search(collection.SearchOptions{Query: query, Filter: filter, Limit: limit, Literal: literal, WithBody: withBody})
			if err != nil {
				return err
			}
			if opts.json {
				return opts.write(cmd.OutOrStdout(), result, "")
			}
			var rows [][]string
			for _, doc := range result.Documents {
				rows = append(rows, []string{doc.ID, terminalCell(doc.Frontmatter["title"].(string))})
			}
			return writeTerminal(cmd.OutOrStdout(), documentTable(cmd.OutOrStdout(), rows, result.Diagnostics, result.HasMore, "matches"))
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum matching documents; 0 returns all")
	cmd.Flags().BoolVar(&literal, "literal", false, "Match a literal case-insensitive substring instead of ranked full text")
	cmd.Flags().BoolVar(&withBody, "with-body", false, "Include Markdown bodies in JSON matches (requires --json)")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Match a frontmatter.data field as KEY=VALUE (repeatable)")
	return cmd
}

func parseSearchFilters(filters []string) (map[string]any, error) {
	data := map[string]any{}
	for _, filter := range filters {
		key, raw, ok := strings.Cut(filter, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("--filter must be KEY=VALUE with a nonempty data field name")
		}
		if _, exists := data[key]; exists {
			return nil, fmt.Errorf("duplicate --filter key %q", key)
		}
		value, err := jsonschema.UnmarshalJSON(strings.NewReader(raw))
		if err != nil {
			if strings.HasPrefix(raw, `"`) || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
				return nil, fmt.Errorf("invalid JSON value for --filter %q: %w", key, err)
			}
			value = raw
		}
		data[key] = value
	}
	return data, nil
}
