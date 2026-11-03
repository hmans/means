package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

func newAppendCommand(opts *options) *cobra.Command {
	var input, hash, bodyFile string
	cmd := &cobra.Command{
		Use:   "append <id> [text]",
		Short: "Append text to a document body",
		Long: `Pass text as one argument, or use --body-file FILE|-.
Alternatively, pass --input FILE|- with {"body":"text"}.
These input modes are exclusive. A dash reads stdin. Text is appended exactly;
no spaces, separators, or newlines are added. Include any desired newlines in
the text. Empty text leaves the body unchanged. Frontmatter and filenames stay.
Means serializes appends per collection, waiting up to five seconds for its lock,
and checks for changes during saving. No prior read or hash is required. To
require a specific version, pass --hash HASH from show --json, or add hash to
the JSON input. A supplied hash must be 64 lowercase hexadecimal characters.
The result must satisfy the schema. Stale hashes and enabled embedding failures
prevent saving. On conflict, read again and reconcile before retrying. A
concurrent external write can still race the final hash check and replacement.
Repeating an append without a hash adds the text again.
Text returns the ID; JSON returns id, hash, frontmatter, body. Errors exit 1.`,
		Example: `  means append <id> "More text." --path ./memory
  means append <id> --body-file ./addition.md --path ./memory
  means append <id> --path ./memory --input - --json <<'JSON'
  {"body":"\n\nMore text.\n"}
JSON

  # Require a specific version
  means show <id> --path ./memory --json
  means append <id> "More text." --hash <hash> --path ./memory`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("hash") && hash == "" {
				return fmt.Errorf("--hash must not be empty; omit it to append to the latest version")
			}
			var text string
			if cmd.Flags().Changed("input") {
				if len(args) != 1 {
					return fmt.Errorf("text and --input are mutually exclusive")
				}
				var data []byte
				var err error
				if input == "-" {
					data, err = io.ReadAll(cmd.InOrStdin())
				} else {
					data, err = os.ReadFile(input)
				}
				if err != nil {
					return fmt.Errorf("read append input: %w", err)
				}
				hash, text, err = parseAppendInput(data)
				if err != nil {
					return err
				}
			} else {
				if cmd.Flags().Changed("body-file") {
					if len(args) != 1 {
						return fmt.Errorf("text and --body-file are mutually exclusive")
					}
					var data []byte
					var err error
					if bodyFile == "-" {
						data, err = io.ReadAll(cmd.InOrStdin())
					} else {
						data, err = os.ReadFile(bodyFile)
					}
					if err != nil {
						return fmt.Errorf("read append body: %w", err)
					}
					text = string(data)
				} else {
					if len(args) != 2 {
						return fmt.Errorf("supply text or --body-file FILE|-")
					}
					text = args[1]
				}
			}
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			doc, err := c.WithEmbeddingProgress(opts.embeddingProgress(cmd)).Append(cmd.Context(), args[0], hash, text)
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), doc, doc.ID+"\n")
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "Append JSON with body and optional hash from FILE, or - for stdin")
	cmd.Flags().StringVar(&hash, "hash", "", "Require this content hash from show --json (optional)")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read text to append from FILE, or - for stdin")
	cmd.MarkFlagsMutuallyExclusive("input", "hash")
	cmd.MarkFlagsMutuallyExclusive("input", "body-file")
	return cmd
}

func parseAppendInput(data []byte) (string, string, error) {
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return "", "", fmt.Errorf("parse append input JSON: %w", err)
	}
	envelope, ok := value.(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("append input must be an object")
	}
	for key := range envelope {
		if key != "hash" && key != "body" {
			return "", "", fmt.Errorf("unknown append field %q", key)
		}
	}
	var hash string
	if value, exists := envelope["hash"]; exists {
		hash, ok = value.(string)
		if !ok || hash == "" {
			return "", "", fmt.Errorf("hash must be a nonempty string; omit it to append to the latest version")
		}
	}
	body, ok := envelope["body"].(string)
	if !ok {
		return "", "", fmt.Errorf("body is required and must be a string")
	}
	return hash, body, nil
}
