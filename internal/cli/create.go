package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

func newCreateCommand(opts *options) *cobra.Command {
	var input string
	var title, bodyFile string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Validate and store a new Markdown document",
		Long: `Create a document with --title TITLE and optional --body-file FILE|-.
This uses empty application data. If the schema requires data fields, use
--input FILE|- with {"frontmatter":{"title":"Title","data":{}},"body":"Markdown"}.
Do not combine these input modes. A dash reads stdin. Means validates the schema
and generates the ID. Text returns the ID; --json returns the document and hash.
Enabled embeddings send title/body to the provider; failures prevent creation.
Creation is not idempotent: retrying a successful create makes another document.
Custom fields belong under data; never supply an ID. Data must satisfy MEANS.md.
JSON returns {"id":"...","hash":"...","frontmatter":{...},"body":"..."}.
Errors exit 1.`,
		Example: `  # Use the default data schema (optional tags)
  means create --path ./memory --input - --json <<'JSON'
  {
    "frontmatter": {"title": "Use Go for means", "data": {"tags": ["decision"]}},
    "body": "We chose Go for single-binary distribution.\n"
  }
JSON

  # Read an envelope from a file
  means create --path ./memory --input ./document.json --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" && !cmd.Flags().Changed("title") {
				return fmt.Errorf("supply --title TITLE or --input FILE|-; --body-file requires --title")
			}
			var data []byte
			var err error
			var frontmatter map[string]any
			var body string
			if cmd.Flags().Changed("title") {
				frontmatter = map[string]any{"title": title, "data": map[string]any{}}
				if cmd.Flags().Changed("body-file") {
					if bodyFile == "-" {
						data, err = io.ReadAll(cmd.InOrStdin())
					} else {
						data, err = os.ReadFile(bodyFile)
					}
					if err != nil {
						return fmt.Errorf("read body: %w", err)
					}
					body = string(data)
				}
			} else {
				if input == "-" {
					data, err = io.ReadAll(cmd.InOrStdin())
				} else {
					data, err = os.ReadFile(input)
				}
				if err != nil {
					return fmt.Errorf("read document input: %w", err)
				}
				frontmatter, body, err = parseDocumentInput(data)
				if err != nil {
					return err
				}
			}
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			doc, err := c.WithEmbeddingProgress(opts.embeddingProgress(cmd)).Create(frontmatter, body)
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), doc, doc.ID+"\n")
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "Document JSON envelope from FILE, or - for stdin (alternative to --title)")
	cmd.Flags().StringVar(&title, "title", "", "Create a document with this title and empty application data")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read the Markdown body from FILE, or - for stdin (requires --title)")
	cmd.MarkFlagsMutuallyExclusive("input", "title")
	cmd.MarkFlagsMutuallyExclusive("input", "body-file")
	return cmd
}

func parseDocumentInput(data []byte) (map[string]any, string, error) {
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("parse document input JSON: %w", err)
	}
	envelope, ok := value.(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("input must be an object with frontmatter and optional body fields")
	}
	for key := range envelope {
		if key != "frontmatter" && key != "body" {
			return nil, "", fmt.Errorf("unknown input field %q; application fields belong inside frontmatter.data", key)
		}
	}
	frontmatter, ok := envelope["frontmatter"].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("frontmatter is required and must be an object")
	}
	body := ""
	if value, exists := envelope["body"]; exists {
		body, ok = value.(string)
		if !ok {
			return nil, "", fmt.Errorf("body must be a string")
		}
	}
	return frontmatter, body, nil
}
