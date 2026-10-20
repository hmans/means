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
  Supply {"frontmatter":{"title":"A document","data":{...}},"body":"Markdown"}.
  frontmatter is required. title must be a nonempty string. data must be an object;
  omit it to use {} if the collection schema permits. body defaults to "".
  Means owns the top-level frontmatter fields. id is generated; do not supply it.
  Application fields belong under data, which alone is validated by schema.json.
  Unknown envelope and top-level frontmatter fields are rejected.

Storage:
  Validate before writing. Generate a collection-scoped 5-character random ID
  using lowercase letters and digits, and store it in frontmatter.id.
  Write <id>-<slug-generated-from-title>.md. Slugs are lowercase letters/digits
  separated by hyphens, bounded to 80 bytes, with "document" as the fallback.
  IDs remain valid after filename changes or moves into visible subdirectories.
  Existing files are never overwritten. Generated collisions retry, using a
  temporary exclusive allocation file to coordinate concurrent means creators.
  Direct edits or Git merges can introduce duplicates; read/update reject them
  and list reports them. Hidden paths and symlinks are excluded from discovery.
  Store frontmatter as YAML between --- lines, followed immediately by the body.
  Body bytes and trailing newlines are preserved. JSON number precision is retained.

Output:
  Text: the generated ID followed by a newline, suitable for passing to read.
  --json: {"id":"...","hash":"...","frontmatter":{...},"body":"..."} on stdout, exit 0.
  hash is SHA-256 of the exact stored bytes, usable as the precondition for update.
  Errors: stderr, exit 1; with --json, {"error":"..."}. Schema errors identify the
  failing field or constraint. JSON output is formatted with two-space indentation.
  Retrying a successful create generates another document; creation is not idempotent.`,
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
