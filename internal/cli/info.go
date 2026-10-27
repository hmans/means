package cli

import (
	"encoding/json"
	"fmt"

	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

func newInfoCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use: "info", Short: "Show collection identity, schema, and workflow",
		Long:    "Read the selected collection's path, title, description, schema, and complete Markdown instructions. Requires a valid MEANS.md. JSON returns path, title, description, schema, and instructions; instructions preserve the exact workflow body.",
		Example: "  means info --path ./issues\n  means info --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			result := struct {
				Path   string          `json:"path"`
				Schema json.RawMessage `json:"schema"`
				*collection.Description
			}{c.Path, c.Schema, c.Describe()}
			schema, err := json.MarshalIndent(result.Schema, "", "  ")
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), result, fmt.Sprintf("%s\n%s\nPath: %s\n\nSchema:\n%s\n\nWorkflow:\n%s", result.Title, result.Description.Description, result.Path, schema, result.Instructions))
		},
	}
}
