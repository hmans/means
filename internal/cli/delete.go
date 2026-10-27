package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

func newDeleteCommand(opts *options) *cobra.Command {
	var input string
	var suppliedHash string
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Permanently delete a document if its content hash still matches",
		Long: `Permanently delete a document using its ID and current hash from show --json.
Pass --hash HASH or --input FILE|- containing {"hash":"<current hash>"}.
These options are exclusive. Stale hashes fail. There is no undo, force, or
confirmation prompt. Directories and collection configuration are kept.
A concurrent external write can still race the final hash check and deletion.
Text returns the deleted ID; JSON returns {"id":"..."}. Errors exit 1.`,
		Example: `  means read <id> --path ./memory --json
  means delete <id> --path ./memory --input - --json <<'JSON'
  {"hash":"<hash from read>"}
JSON`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" && !cmd.Flags().Changed("hash") {
				return fmt.Errorf("supply --hash HASH or --input FILE|-")
			}
			var data []byte
			var err error
			hash := suppliedHash
			if !cmd.Flags().Changed("hash") {
				if input == "-" {
					data, err = io.ReadAll(cmd.InOrStdin())
				} else {
					data, err = os.ReadFile(input)
				}
				if err != nil {
					return fmt.Errorf("read delete input: %w", err)
				}
				hash, err = parseDeleteInput(data)
				if err != nil {
					return err
				}
			}
			c, err := opts.loadCollection()
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
	cmd.Flags().StringVar(&input, "input", "", "Delete JSON with content hash from FILE, or - for stdin (alternative to --hash)")
	cmd.Flags().StringVar(&suppliedHash, "hash", "", "Current content hash from show --json or read --json")
	cmd.MarkFlagsMutuallyExclusive("input", "hash")
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
