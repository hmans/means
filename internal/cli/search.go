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
	var filters filterFlags
	var literal bool
	var semantic bool
	var minScore float64
	var withBody bool
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Find documents by ranked full text and application data",
		Long: `Find documents by words in their title/body, ranked by relevance.
Use --literal for a case-insensitive substring or --semantic for embedding similarity;
these modes are exclusive. Semantic search requires configured embeddings and a
nonempty query, sends the query to the provider, and skips stale/missing vectors.
Run embed to refresh vectors. --min-score applies only to semantic search.
Combine --filter KEY=VALUE and RFC 3339 datetime flags with AND before --limit.
Omit the query to filter only by data. Default limit is 20; 0 returns all.
--json includes metadata and excerpts; --with-body adds bodies. Inspect diagnostics
for malformed/duplicate documents; finding no matches still exits successfully.
JSON returns documents, diagnostics, has_more; hashes are omitted. Invalid arguments
and collection-level failures exit 1. Text search requires all words to match.
Filter keys name fields directly under data; objects match supplied keys recursively,
arrays match in full, and a missing field never matches. Quote JSON in shell arguments.
Datetimes need timezone offsets; before/after are exclusive, on-or variants inclusive.`,
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
			filter, dateTime, err := filters.parse()
			if err != nil {
				return err
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
	filters.bind(cmd)
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
