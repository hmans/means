package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hmans/means/internal/collection"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

func newSearchCommand(opts *options) *cobra.Command {
	var limit int
	var filters []string
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Find documents by literal text and application data",
		Long: `Search visible documents in the collection selected by --path.

The optional query is a case-insensitive literal substring of the title or
Markdown body. It is one phrase, not a regular expression or a set of words.
Omit it to filter by data alone. Results are sorted by relative filename. A
text match includes its field (title or body) and a short excerpt. Results
include frontmatter but omit bodies and hashes; read by ID for full content.

Repeat --filter KEY=VALUE to match fields directly under frontmatter.data.
All filters combine with AND. Plain values such as priority=normal are strings;
JSON values such as count=2, active=true, tags=["search"], or nested={"a":1}
retain their types. Quote the entire KEY=VALUE argument in your shell when it
contains JSON quotes or spaces. Nested objects match recursively by supplied
keys; arrays must match in full. A missing field never matches, even for null.
Duplicate filter keys are rejected.

--limit defaults to 20 matching documents; 0 returns all. has_more reports
whether further matches exist. Malformed documents and duplicate IDs are
omitted from matches and reported separately in diagnostics, regardless of
query or result limit. Each diagnostic has an ID when readable, relative path,
and error. Application data that fails the current schema remains searchable.
Hidden paths, symlinks, and root MEANS.md are skipped.
Subdirectories containing means.schema.json are separate collections and are skipped,
even if their schema is invalid.

--json returns {"documents":[...],"diagnostics":[],"has_more":false}.
Text output shows IDs, metadata, and excerpts, followed by diagnostics.
Search exits 0 even with per-document diagnostics. Invalid arguments or
collection-level errors exit 1 and write stderr, as {"error":"..."} in JSON
mode. Search only reads files; it does not repair or rewrite documents.`,
		Example: `  means search schema --path ./issues --json
  means search --path ./issues --filter priority=normal --filter type=feature --json
  means search memory --filter 'tags=["important"]' --limit 0 --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			result, err := c.Search(collection.SearchOptions{Query: query, Filter: filter, Limit: limit})
			if err != nil {
				return err
			}
			var output strings.Builder
			for _, doc := range result.Documents {
				metadata, err := json.Marshal(doc.Frontmatter)
				if err != nil {
					return err
				}
				fmt.Fprintf(&output, "%s\t%s", doc.ID, metadata)
				if doc.Match != nil {
					fmt.Fprintf(&output, "\t%s: %s", doc.Match.Field, doc.Match.Excerpt)
				}
				output.WriteByte('\n')
			}
			for _, diagnostic := range result.Diagnostics {
				fmt.Fprintf(&output, "%s\t%s\tERROR: %s\n", diagnostic.ID, diagnostic.Path, diagnostic.Error)
			}
			if result.HasMore {
				output.WriteString("More matches available; increase --limit or use --limit 0.\n")
			}
			return opts.write(cmd.OutOrStdout(), result, output.String())
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum matching documents; 0 returns all")
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
