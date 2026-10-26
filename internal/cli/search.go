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
	dateTimeFlags := map[string]*[]string{}
	var literal bool
	var semantic bool
	var minScore float64
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
Use --semantic for opt-in embedding search: embed only the query and rank stored
document vectors by cosine similarity (ID breaks ties). Search never refreshes
or removes document vectors. Missing or stale vectors are skipped; run means embed
to refresh them. Only the query is sent to the configured embedding endpoint.
Requires embeddings in the machine-local Means user config and API credentials; see means embed
--help. A nonempty query is required; --literal and --semantic are exclusive.
Semantic results include score and omit match. --min-score defaults to 0.3 and
keeps scores at or above the threshold before applying --limit. Values must be
finite and between -1 and 1; -1 includes all scores. Requires --semantic.
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

Use --before KEY=TIMESTAMP, --after KEY=TIMESTAMP, --on-or-before KEY=TIMESTAMP,
or --on-or-after KEY=TIMESTAMP for datetime fields directly under data. Repeat
flags to bound a range. All datetime and exact-value filters combine with AND,
in every search mode, before limits. Before/after are exclusive; on-or variants
include equal instants. Values must be RFC 3339 timestamps with a timezone,
e.g. 2026-10-26T09:00:00Z or 2026-10-26T10:00:00+01:00 (the same instant).
Up to 9 fractional second digits are supported; leap seconds, date-only values,
and timestamps without a timezone are unsupported. Invalid filter values fail;
missing, non-string, or invalid document values do not match. Field names are
literal keys, not paths. Datetimes remain strings in stored data and JSON;
ordinary --filter equality remains exact. No schema annotation is required.

--limit defaults to 20 matching documents; 0 returns all; negative values fail.
has_more reports whether further matches exist. Malformed documents and duplicate IDs are
omitted from matches and reported separately in diagnostics, regardless of
query or result limit. Each diagnostic has an ID when readable, relative path,
and error. Application data that fails the current schema remains searchable.
Hidden paths, symlinks, and root MEANS.md are skipped.
Subdirectories containing MEANS.md are separate collections and are skipped,
even if their schema is invalid.

--json returns {"documents":[...],"diagnostics":[],"has_more":false}.
Each document has id and frontmatter. Text matches also have a match object,
for example {"field":"title","excerpt":"..."}. field is "title" or "body";
data-only results omit match.
No matches returns an empty documents array and exits 0; inspect diagnostics too.
Text output is a colored ID/title table (ID/score/title for semantic search),
followed by diagnostics and limit notices. Scores are shown to three decimals;
filtering uses full precision, not the displayed rounded value.
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
  means search openai --semantic --min-score 0.3 --path ./issues
  means search "deployment timeout" --literal --path ./memory --json
  means search --path ./issues --filter priority=normal --filter type=feature --json
  means search --after created_at=2026-10-01T00:00:00Z --before created_at=2026-11-01T00:00:00Z --json
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
			var dateTime []collection.DateTimeFilter
			for _, flag := range []string{"before", "after", "on-or-before", "on-or-after"} {
				for _, raw := range *dateTimeFlags[flag] {
					field, value, ok := strings.Cut(raw, "=")
					if !ok || field == "" {
						return fmt.Errorf("--%s must be KEY=TIMESTAMP with a nonempty data field name", flag)
					}
					dateTime = append(dateTime, collection.DateTimeFilter{Field: field, Op: strings.ReplaceAll(flag, "-", "_"), Value: value})
				}
			}
			query := ""
			if len(args) == 1 {
				query = args[0]
			}
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			var threshold *float64
			if cmd.Flags().Changed("min-score") {
				threshold = &minScore
			}
			result, err := c.Search(collection.SearchOptions{Query: query, Filter: filter, DateTime: dateTime, Limit: limit, Literal: literal, Semantic: semantic, WithBody: withBody, MinScore: threshold})
			if err != nil {
				return err
			}
			if opts.json {
				return opts.write(cmd.OutOrStdout(), result, "")
			}
			var rows [][]string
			for _, doc := range result.Documents {
				row := []string{doc.ID}
				if semantic {
					row = append(row, fmt.Sprintf("%.3f", *doc.Score))
				}
				rows = append(rows, append(row, terminalCell(doc.Frontmatter["title"].(string))))
			}
			headers := []string{"ID", "TITLE"}
			if semantic {
				headers = []string{"ID", "SCORE", "TITLE"}
			}
			return writeTerminal(cmd.OutOrStdout(), documentTable(cmd.OutOrStdout(), rows, result.Diagnostics, result.HasMore, "matches", headers...))
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum matching documents; 0 returns all")
	cmd.Flags().BoolVar(&literal, "literal", false, "Match a literal case-insensitive substring instead of ranked full text")
	cmd.Flags().BoolVar(&semantic, "semantic", false, "Rank by embedding similarity (requires opt-in configuration and API credentials)")
	cmd.Flags().Float64Var(&minScore, "min-score", collection.DefaultSemanticMinScore, "Minimum cosine similarity, inclusive, from -1 to 1 (requires --semantic; -1 includes all scores)")
	cmd.Flags().BoolVar(&withBody, "with-body", false, "Include Markdown bodies in JSON matches (requires --json)")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Match a frontmatter.data field as KEY=VALUE (repeatable)")
	for _, flag := range []string{"before", "after", "on-or-before", "on-or-after"} {
		dateTimeFlags[flag] = cmd.Flags().StringArray(flag, nil, "Compare a data datetime as KEY=TIMESTAMP (RFC 3339; repeatable)")
	}
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
