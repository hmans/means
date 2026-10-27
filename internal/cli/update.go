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

func newUpdateCommand(opts *options) *cobra.Command {
	var input string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Patch a document only if its content hash still matches",
		Long: `Read the document with show --json or read --json, then patch it with --input FILE|-:
{"hash":"<current hash>","frontmatter":{"data":{"status":"done"}}}.
Omitted fields stay; null deletes metadata keys; arrays replace. A supplied body
replaces the full body. Stale hashes and invalid data fail without saving.
Enabled embedding failures also prevent saving. Use edit for an editor workflow.
IDs are immutable; title changes preserve filenames. Null deletes keys and cannot
set literal null values (use edit for that). On conflict, read again and reconcile.
A concurrent external write can still race the final hash check and replacement.
Text returns the ID; JSON returns id, hash, frontmatter, body. Errors exit 1.`,
		Example: `  means read <id> --path ./memory --json
  means update <id> --path ./memory --input - --json <<'JSON'
  {
    "hash": "<hash from read>",
    "frontmatter": {"title": "Corrected title", "data": {"tags": ["decision"]}},
    "body": "Updated Markdown body.\n"
  }
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
				return fmt.Errorf("read update input: %w", err)
			}
			hash, patch, err := parseUpdateInput(data)
			if err != nil {
				return err
			}
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			doc, err := c.WithEmbeddingProgress(opts.embeddingProgress(cmd)).Update(args[0], hash, patch)
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), doc, doc.ID+"\n")
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "Update JSON with hash and patch fields from FILE, or - for stdin (required)")
	return cmd
}

func parseUpdateInput(data []byte) (string, collection.Patch, error) {
	var patch collection.Patch
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return "", patch, fmt.Errorf("parse update input JSON: %w", err)
	}
	envelope, ok := value.(map[string]any)
	if !ok {
		return "", patch, fmt.Errorf("update input must be an object")
	}
	for key := range envelope {
		if key != "hash" && key != "frontmatter" && key != "body" {
			return "", patch, fmt.Errorf("unknown update field %q", key)
		}
	}
	hash, ok := envelope["hash"].(string)
	if !ok {
		return "", patch, fmt.Errorf("hash is required; read the document first")
	}
	if value, exists := envelope["frontmatter"]; exists {
		patch.Frontmatter, ok = value.(map[string]any)
		if !ok {
			return "", patch, fmt.Errorf("frontmatter patch must be an object")
		}
	}
	if value, exists := envelope["body"]; exists {
		body, ok := value.(string)
		if !ok {
			return "", patch, fmt.Errorf("body must be a string")
		}
		patch.Body = &body
	}
	return hash, patch, nil
}
