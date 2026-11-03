package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

func newInstructionsCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use: "instructions", Short: "Manage the collection workflow body",
		Long:    "Use info to read the MEANS.md body and its hash, then instructions set to replace it.",
		Example: "  means info --json\n  means instructions set --input instructions.json --json",
		Args:    cobra.NoArgs,
		RunE:    func(cmd *cobra.Command, _ []string) error { return opts.writeHelp(cmd) },
	}
	var input string
	set := &cobra.Command{
		Use: "set", Short: "Replace the MEANS.md body using its current hash",
		Long: `Read info --json, then supply --input FILE|- with JSON:
{"hash":"<hash from info>","instructions":"Replacement Markdown.\n"}.
Instructions replace the full MEANS.md body; an empty string clears it.
Preserves exact frontmatter bytes and file permissions. Requires valid collection
configuration, but does not validate or change documents. A stale hash fails;
call info again and reconcile before retrying. Observed concurrent changes reject
replacement; external writes can still race the final check and rename.
Text confirms the update. JSON returns hash, title, description, instructions.
Errors exit 1 and go to stderr.`,
		Example: `  means info --path ./memory --json
  means instructions set --path ./memory --input instructions.json --json
  means instructions set --input - --json <<'JSON'
  {"hash":"<hash from info>","instructions":"Search before creating notes.\n"}
JSON`,
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
				return fmt.Errorf("read instructions input: %w", err)
			}
			hash, instructions, err := parseInstructionsInput(data)
			if err != nil {
				return err
			}
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			result, err := c.SetInstructions(cmd.Context(), hash, instructions)
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), result, "Updated collection instructions.\n")
		},
	}
	set.Flags().StringVar(&input, "input", "", "JSON with hash and instructions from FILE, or - for stdin (required)")
	cmd.AddCommand(set)
	return cmd
}

func parseInstructionsInput(data []byte) (string, string, error) {
	if !utf8.Valid(data) {
		return "", "", fmt.Errorf("instructions input must be valid UTF-8")
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return "", "", fmt.Errorf("parse instructions input JSON: %w", err)
	}
	input, ok := value.(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("instructions input must be an object")
	}
	for key := range input {
		if key != "hash" && key != "instructions" {
			return "", "", fmt.Errorf("unknown instructions field %q", key)
		}
	}
	hash, hashOK := input["hash"].(string)
	instructions, instructionsOK := input["instructions"].(string)
	if !hashOK || !instructionsOK {
		return "", "", fmt.Errorf("hash and instructions are required strings; call info first")
	}
	return hash, instructions, nil
}
