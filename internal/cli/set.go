package cli

import (
	"fmt"
	"strings"

	"github.com/hmans/means/collection"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

func newSetCommand(opts *options) *cobra.Command {
	var title, hash string
	cmd := &cobra.Command{
		Use:   "set <id> [KEY=VALUE | KEY+=VALUE | KEY-=VALUE]...",
		Short: "Change data fields or the title without supplying a hash",
		Long: `Change application data fields in one step. KEY names a field under data;
use dots for nested objects (owner.name). VALUE is JSON when it parses as JSON,
otherwise a string; quote JSON strings that look like numbers or booleans.
KEY=VALUE assigns, creating missing parent objects; KEY=null deletes the field.
KEY+=VALUE appends to an array unless an equal element exists, creating it if
missing. KEY-=VALUE removes equal elements. --title replaces the title.
Edits apply in order to the latest version. Pass --hash HASH from show --json to
require a specific version. Set shares append's per-collection lock, so
concurrent set and append calls do not lose changes; other writes can still
race the final replacement. The result must satisfy the schema; invalid data and
enabled embedding failures prevent saving. Use update for body changes.
Text returns the ID; JSON returns id, hash, frontmatter, body. Errors exit 1.`,
		Example: `  means set <id> status=done --path ./issues
  means set <id> priority=high tags+=urgent tags-=later --path ./issues
  means set <id> owner.name=Ada estimate=3 'code="042"' --path ./issues
  means set <id> due=null --title "Renamed issue" --path ./issues --json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("hash") && hash == "" {
				return fmt.Errorf("--hash must not be empty; omit it to edit the latest version")
			}
			var edits []collection.FieldEdit
			if cmd.Flags().Changed("title") {
				edits = append(edits, collection.FieldEdit{Path: []string{"title"}, Value: title})
			}
			for _, arg := range args[1:] {
				edit, err := parseFieldEdit(arg)
				if err != nil {
					return err
				}
				edits = append(edits, edit)
			}
			if len(edits) == 0 {
				return fmt.Errorf("supply at least one KEY=VALUE edit or --title")
			}
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			doc, err := c.WithEmbeddingProgress(opts.embeddingProgress(cmd)).Set(cmd.Context(), args[0], hash, edits...)
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), doc, doc.ID+"\n")
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Replace the document title")
	cmd.Flags().StringVar(&hash, "hash", "", "Require this current content hash from show --json")
	return cmd
}

func parseFieldEdit(arg string) (collection.FieldEdit, error) {
	key, raw, ok := strings.Cut(arg, "=")
	if !ok || key == "" {
		return collection.FieldEdit{}, fmt.Errorf("edit %q must be KEY=VALUE, KEY+=VALUE, or KEY-=VALUE", arg)
	}
	op := collection.FieldAssign
	if strings.HasSuffix(key, "+") {
		op, key = collection.FieldAdd, strings.TrimSuffix(key, "+")
	} else if strings.HasSuffix(key, "-") {
		op, key = collection.FieldRemove, strings.TrimSuffix(key, "-")
	}
	path := strings.Split(key, ".")
	for _, part := range path {
		if part == "" {
			return collection.FieldEdit{}, fmt.Errorf("edit %q has an empty field name", arg)
		}
	}
	value, err := jsonschema.UnmarshalJSON(strings.NewReader(raw))
	if err != nil {
		if strings.HasPrefix(raw, `"`) || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
			return collection.FieldEdit{}, fmt.Errorf("invalid JSON value in edit %q: %w", arg, err)
		}
		value = raw
	}
	return collection.FieldEdit{Path: append([]string{"data"}, path...), Op: op, Value: value}, nil
}
