package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

// ErrSchemaRejected means the diagnostic error response has already been written.
var ErrSchemaRejected = errors.New("schema installation rejected")

func newSchemaCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use: "schema", Short: "Manage the collection schema",
		Long:    "Manage MEANS.md in the collection selected by --path. Use schema set to validate and install a proposed JSON Schema.",
		Example: "  means schema set --path ./memory --input new-schema.json --json",
	}
	var input string
	var force bool
	set := &cobra.Command{
		Use: "set", Short: "Validate and install a proposed collection schema",
		Long: `Validate and install JSON Schema from --input FILE|- into MEANS.md.
Check existing documents before replacing the schema. --force allows document
violations, but never invalid schemas or filesystem errors. Documents are not
migrated. Malformed MEANS.md YAML requires direct editing. Run doctor afterward.
JSON returns installed, healthy, checked, diagnostics. Rejection exits 1 and reports
diagnostics on stderr. Observed concurrent MEANS.md edits reject replacement;
external writes can still race installation.`,
		Example: `  means schema set --path ./memory --input new-schema.json --json
  means schema set --path ./memory --input - --force --json < new-schema.json
  means doctor --path ./memory --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" {
				return fmt.Errorf("--input is required; supply a file path or - for stdin")
			}
			var data []byte
			var err error
			if input == "-" {
				data, err = io.ReadAll(cmd.InOrStdin())
			} else {
				data, err = os.ReadFile(input)
			}
			if err != nil {
				return fmt.Errorf("read schema input: %w", err)
			}
			result, err := collection.SetSchema(opts.path, data, force)
			if err != nil && !errors.Is(err, collection.ErrSchemaIncompatible) {
				return err
			}
			var text strings.Builder
			if result.Installed {
				text.WriteString("Installed schema.\n")
			} else {
				text.WriteString("Schema not installed; use --force to allow document violations.\n")
			}
			fmt.Fprintf(&text, "Checked %d documents.\n", result.Checked)
			for _, d := range result.Diagnostics {
				fmt.Fprintf(&text, "%s\t%s\tERROR: %s\n", d.ID, d.Path, d.Error)
			}
			if err != nil {
				response := struct {
					Error string `json:"error"`
					*collection.SchemaSetResult
				}{err.Error(), result}
				if err := opts.write(cmd.ErrOrStderr(), response, text.String()); err != nil {
					return err
				}
				return ErrSchemaRejected
			}
			return opts.write(cmd.OutOrStdout(), result, text.String())
		},
	}
	set.Flags().StringVar(&input, "input", "", "Read proposed JSON Schema from FILE, or - for stdin (required)")
	set.Flags().BoolVar(&force, "force", false, "Install despite document violations; still report diagnostics")
	cmd.AddCommand(set)
	return cmd
}
