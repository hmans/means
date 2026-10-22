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
		Long:    "Manage means.schema.json in the collection selected by --path. Use schema set to validate and install a proposed JSON Schema.",
		Example: "  means schema set --path ./memory --input new-schema.json --json",
	}
	var input string
	var force bool
	set := &cobra.Command{
		Use: "set", Short: "Validate and install a proposed collection schema",
		Long: `Validate and replace means.schema.json in the existing collection at --path.

--input FILE reads a proposed JSON Schema; --input - reads stdin until EOF.
Relative input paths resolve against the working directory, independently of
--path. Supply the schema itself, not a wrapper. Draft 2020-12 is the default.
Internal references are supported; external references are not. Invalid or
unsupported schemas always fail, even with --force. Use init for a new collection.
An existing invalid schema can be replaced, but it must be a regular file.

Check all visible documents against the proposed schema, respecting nested
collection boundaries. This checks current data, not abstract compatibility with
the old schema or future callers. A stricter schema can pass on an empty collection.
Malformed envelopes, duplicate IDs, and data violations block installation.
--force permits these document diagnostics but still reports them. It does not
bypass schema validation or collection-level filesystem errors.

No documents are changed. Installation preserves the supplied schema bytes and
existing permission bits, using a same-directory temporary file and replacement.
There are no locks: concurrent document edits can invalidate the check, and a
concurrent schema edit can race the final replacement. Observed schema changes
are rejected. Run doctor after installation and let the agent or user fix data.

Success exits 0 and writes stdout. --json returns
{"installed":true,"healthy":true,"checked":0,"diagnostics":[]}.
Forced installation can return healthy:false with diagnostics (id, path, error).
Document rejection exits 1, leaves the schema unchanged, and writes stderr:
{"error":"...","installed":false,"healthy":false,"checked":1,"diagnostics":[...]}.
Other failures exit 1 with {"error":"..."} on stderr. Text output reports whether
the schema was installed, the checked count, and every diagnostic.
MCP calls reload the schema; restart MCP to refresh advertised collection metadata.`,
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
