package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/hmans/means/internal/collection"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

func newDeleteCommand(opts *options) *cobra.Command {
	var input string
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Permanently delete a document if its content hash still matches",
		Long: `Permanently remove one document from --path (default: current directory).

Read the document first with "means read <id> --json". Pass its stored ID, not a
filename, and supply {"hash":"<64 lowercase hexadecimal SHA-256 characters>"}
through --input FILE or --input - for stdin. File paths are relative to the
working directory. Input must be one JSON object; unknown fields are rejected.

Lookup scans visible subdirectories but stops at means.schema.json boundaries.
Hidden paths, symlinks, and root MEANS.md are excluded. Missing or duplicate IDs,
malformed document envelopes, and stale or invalid hashes fail without deletion.
Existing application data need not match the current schema, but the selected
collection must have a valid means.schema.json.

Only the matched document file is removed. Directories, collection configuration,
and unrelated files remain. There is no trash, undo, force option, or confirmation
prompt. A repeated deletion fails because the document is missing. Use a data
field such as status for archival instead of deletion when retention is needed.
There are no locks: a concurrent write between the hash check and removal can
still be deleted. On conflict, read again and review before retrying.

Success exits 0. Text output is the deleted ID followed by a newline.
--json returns {"id":"..."}. Errors exit 1 and go to stderr, formatted as
{"error":"..."} with --json. No document content or new hash is returned.`,
		Example: `  means read <id> --path ./memory --json
  means delete <id> --path ./memory --input - --json <<'JSON'
  {"hash":"<hash from read>"}
JSON`,
		Args: cobra.ExactArgs(1),
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
				return fmt.Errorf("read delete input: %w", err)
			}
			hash, err := parseDeleteInput(data)
			if err != nil {
				return err
			}
			c, err := collection.Load(opts.path)
			if err != nil {
				return err
			}
			result, err := c.Delete(args[0], hash)
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), result, result.ID+"\n")
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "Delete JSON with content hash from FILE, or - for stdin (required)")
	return cmd
}

func parseDeleteInput(data []byte) (string, error) {
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("parse delete input JSON: %w", err)
	}
	envelope, ok := value.(map[string]any)
	if !ok {
		return "", fmt.Errorf("delete input must be an object")
	}
	for key := range envelope {
		if key != "hash" {
			return "", fmt.Errorf("unknown delete field %q", key)
		}
	}
	hash, ok := envelope["hash"].(string)
	if !ok {
		return "", fmt.Errorf("hash is required; read the document first")
	}
	return hash, nil
}
