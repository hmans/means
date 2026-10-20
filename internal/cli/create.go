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

func newCreateCommand(opts *options) *cobra.Command {
	var input string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Validate and store a new Markdown document",
		Long: `Create one document in the collection selected by --path (default: current directory).
Initialize the collection first with "means init" and read its schema.json.

Input:
  --input FILE reads a JSON file relative to the current working directory,
  independently of --path. --input - reads stdin until EOF. Input is required.
  Supply one object: {"frontmatter":{...},"body":"Markdown text"}.
  frontmatter is required and must be an object; {} is allowed if the schema permits.
  body is an optional string, defaulting to "". Other envelope fields are rejected.
  Application fields, including title or id, belong inside frontmatter and are
  governed entirely by schema.json. No application fields are added automatically.

Storage:
  Validate frontmatter before writing. Invalid input or metadata creates no file.
  Assign a random 32-character hexadecimal ID and write <id>.md in the collection.
  The ID is the filename stem; renaming the file changes its ID.
  Store metadata as YAML between --- lines, followed immediately by the body.
  The body's content and trailing newline are preserved. Existing files are never
  overwritten. JSON strings and numbers retain their types through YAML storage.

Output:
  Text: the generated ID followed by a newline, suitable for passing to read.
  --json: {"id":"...","hash":"...","frontmatter":{...},"body":"..."} on stdout, exit 0.
  hash is SHA-256 of the exact stored bytes, usable as the precondition for update.
  Errors: stderr, exit 1; with --json, {"error":"..."}. Schema errors identify the
  failing field or constraint. JSON output is formatted with two-space indentation.
  Retrying a successful create generates another document; creation is not idempotent.`,
		Example: `  # Use the default collection schema (title and optional tags)
  means create --path ./memory --input - --json <<'JSON'
  {
    "frontmatter": {"title": "Use Go for means", "tags": ["decision"]},
    "body": "We chose Go for single-binary distribution.\n"
  }
JSON

  # Read an envelope from a file
  means create --path ./memory --input ./document.json --json`,
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
				return fmt.Errorf("read document input: %w", err)
			}
			frontmatter, body, err := parseDocumentInput(data)
			if err != nil {
				return err
			}
			c, err := collection.Load(opts.path)
			if err != nil {
				return err
			}
			doc, err := c.Create(frontmatter, body)
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), doc, doc.ID+"\n")
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "Document JSON envelope from FILE, or - for stdin (required)")
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
			return nil, "", fmt.Errorf("unknown input field %q; application fields belong inside frontmatter", key)
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
