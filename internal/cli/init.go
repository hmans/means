package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

func newInitCommand(opts *options) *cobra.Command {
	var schemaSource string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a collection with an example or supplied means.schema.json",
		Long: `Initialize a collection in --path (default: the current working directory).

For agent setup, run "means agents" first. After initialization, write MEANS.md
and establish discovery through project instructions, a skill, or MCP. Init creates
storage; it does not install agent integrations or write workflow guidance.

Without --schema, write an example data schema allowing optional tags (an array
of strings), rejecting other data fields. An empty data object is valid.
The schema validates only frontmatter.data. Means separately owns id (generated,
immutable) and title (required, nonempty string). Do not wrap the schema in data;
its properties describe fields inside data. Schema title/description annotations
still identify the collection for MCP discovery.

For an agent-created collection:
  1. Choose application data fields and constraints for the user's intended memory.
  2. Author a standalone JSON Schema and include descriptions explaining fields.
     Set its top-level title and description to identify the collection's purpose
     to MCP clients (separate from fields inside properties).
  3. Supply it using --schema FILE or --schema - to read stdin until EOF.
  4. Inspect the response; correct invalid schemas and retry if needed.

Schema input:
  Supply one JSON Schema value, not a document or a {"schema": ...} wrapper.
  Usually use type: "object", properties, required, and additionalProperties.
  Use enum for fixed choices, items for array entries, and description for meaning.
  JSON Schema Draft 2020-12 is the default when $schema is omitted.
  Internal references such as #/$defs/status work. External files, remote
  references, and custom remote metaschemas are not supported.
  Relative --schema file paths resolve against the current working directory,
  independently of --path. Stdin is read only with --schema -.

Effects:
  Validate the entire schema before creating the target directory or any files.
  Create missing directories and write means.schema.json, preserving supplied bytes.
  Create .gitignore with .means/, or append the rule while preserving existing
  content. Do not duplicate an existing .means/ or /.means/ rule. Symlinked or
  non-regular .gitignore files are rejected. Search caches are created lazily.
  Refuse to overwrite an existing means.schema.json, even if it is invalid.
  Other existing files are left alone. No documents are created or checked.

Responses:
  Success: exit 0; stdout describes the initialized collection.
  With --json: {"path":"/absolute/collection","schema":"/absolute/collection/means.schema.json","next_steps":["means agents"]}.
  Failure: exit 1; stderr explains the error and stdout is empty.
  With --json: {"error":"..."}; schema errors identify the invalid constraint.
  JSON responses use two-space indentation and end with a newline.

After initialization, use "means schema set --help" to validate and replace the
schema, or edit means.schema.json directly. Use create to store documents, list
to discover IDs, read to retrieve them, and doctor to check their current validity.
Schemas describe application data nested in YAML frontmatter; the body is free-form.
Optionally author MEANS.md beside means.schema.json for MCP workflow guidance;
init does not create it. See "means mcp --help" for collection identity and instructions.`,
		Example: `  # Start with the example schema
  means init --path ./memory

  # Install an agent-authored schema from a file
  means init --path ./memory --schema ./memory.schema.json --json

  # Supply a complete schema through stdin (POSIX shell)
  means init --path ./memory --schema - --json <<'JSON'
  {
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "type": "object",
    "properties": {
      "summary": {"type": "string", "description": "A fact worth remembering."},
      "kind": {"enum": ["fact", "decision"], "description": "The type of memory."}
    },
    "required": ["summary", "kind"],
    "additionalProperties": false
  }
JSON`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var schema []byte
			if cmd.Flags().Changed("schema") {
				var err error
				switch schemaSource {
				case "":
					return fmt.Errorf("--schema must be a file path or - for stdin")
				case "-":
					schema, err = io.ReadAll(cmd.InOrStdin())
				default:
					schema, err = os.ReadFile(schemaSource)
				}
				if err != nil {
					return fmt.Errorf("read schema from %q: %w", schemaSource, err)
				}
				if len(schema) == 0 {
					return fmt.Errorf("schema input is empty; supply a JSON Schema")
				}
			}
			c, err := collection.Init(opts.path, schema)
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), struct {
				Path      string   `json:"path"`
				Schema    string   `json:"schema"`
				NextSteps []string `json:"next_steps"`
			}{c.Path, filepath.Join(c.Path, collection.SchemaFilename), []string{"means agents"}}, "Initialized collection at "+c.Path+"\nNext: run means agents to add workflow guidance and agent discovery.\n")
		},
	}
	cmd.Flags().StringVar(&schemaSource, "schema", "", "Read JSON Schema from FILE, or - for stdin (default: example schema)")
	return cmd
}
