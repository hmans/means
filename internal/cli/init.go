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
	var configSource string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a collection with an example or supplied MEANS.md",
		Long: `Initialize a collection in --path (default: the current working directory).

For agent setup, run "means agents" first. Init writes MEANS.md; refine its guidance
and establish discovery through project instructions, a skill, or MCP.

Configuration input:
  --input FILE reads a complete MEANS.md; --input - reads stdin until EOF.
  MEANS.md starts with --- delimited YAML frontmatter. The required schema field
  is a JSON Schema object or boolean expressed in YAML. Optional title and
  description fields must be strings. Unknown fields are rejected.
  Missing or blank title falls back to the directory name; description defaults
  to empty. The Markdown body supplies collection self-description and workflow.
  Use JSON-compatible YAML: string keys, exact numbers, no aliases, merge keys,
  custom tags, or non-finite numbers. The body may be empty.
  --input and --schema are mutually exclusive. Relative input paths resolve
  against the working directory, independently of --path.

Without either input flag, write a directory-derived title, editable guidance,
and an example data schema allowing optional tags (an array of strings),
rejecting other data fields. An empty data object is valid.
The schema validates only frontmatter.data. Means separately owns id (generated,
immutable) and title (required, nonempty string). Do not wrap the schema in data;
its properties describe fields inside data. Collection title/description come from
MEANS.md configuration fields, not annotations inside the schema.

For an agent-created collection:
  1. Choose application data fields and constraints for the user's intended memory.
  2. Author MEANS.md with title, description, schema, and Markdown guidance.
  3. Supply it using --input FILE or --input -; alternatively provide only a
     JSON Schema using --schema FILE or --schema - to generate the rest.
  4. Inspect the response; correct invalid schemas and retry if needed.

Schema input:
  Supply one JSON Schema value, not a document or a {"schema": ...} wrapper.
  Usually use type: "object", properties, required, and additionalProperties.
  Use enum for fixed choices, items for array entries, and description for meaning.
  JSON Schema Draft 2020-12 is the default when $schema is omitted.
  Internal references such as #/$defs/status work. External files, remote
  references, and custom remote metaschemas are not supported.
  Relative --schema file paths resolve against the current working directory,
  independently of --path. Stdin is read only when an input flag is -.

Effects:
  Validate configuration and schema before creating the target directory or files.
  Create missing directories and write MEANS.md. --input preserves supplied bytes;
  --schema serializes the JSON Schema into YAML, keeping key order and exact numbers.
  Create .gitignore with .means/, or append the rule while preserving existing
  content. Do not duplicate an existing .means/ or /.means/ rule. Symlinked or
  non-regular .gitignore files are rejected. Search caches are created lazily.
  Refuse to overwrite an existing MEANS.md, even if it is invalid.
  Other existing files are left alone. No documents are created or checked.

Responses:
  Success: exit 0; stdout describes the initialized collection.
  With --json: {"path":"/absolute/collection","config":"/absolute/collection/MEANS.md","next_steps":["means agents"]}.
  Failure: exit 1; stderr explains the error and stdout is empty.
  With --json: {"error":"..."}; schema errors identify the invalid constraint.
  JSON responses use two-space indentation and end with a newline.

After initialization, use "means schema set --help" to validate and replace the
schema, or edit MEANS.md directly. Use create to store documents, list
to discover IDs, read to retrieve them, and doctor to check their current validity.
Schemas describe application data nested in YAML frontmatter; the body is free-form.
Edit MEANS.md to change identity and guidance. See "means mcp --help" for discovery.`,
		Example: `  # Start with the example schema
  means init --path ./memory

  # Install a complete configuration from a file or stdin
  means init --path ./memory --input ./template/MEANS.md --json
  means init --path ./memory --input - --json < ./template/MEANS.md

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
			if cmd.Flags().Changed("input") {
				if configSource == "" {
					return fmt.Errorf("--input must be a file path or - for stdin")
				}
				var data []byte
				var err error
				if configSource == "-" {
					data, err = io.ReadAll(cmd.InOrStdin())
				} else {
					data, err = os.ReadFile(configSource)
				}
				if err != nil {
					return fmt.Errorf("read configuration: %w", err)
				}
				c, err := collection.InitConfig(opts.path, data)
				if err != nil {
					return err
				}
				return writeInitResult(cmd, opts, c)
			}
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
			return writeInitResult(cmd, opts, c)
		},
	}
	cmd.Flags().StringVar(&schemaSource, "schema", "", "Read JSON Schema from FILE, or - for stdin (default: example schema)")
	cmd.Flags().StringVar(&configSource, "input", "", "Read complete MEANS.md from FILE, or - for stdin")
	cmd.MarkFlagsMutuallyExclusive("schema", "input")
	return cmd
}

func writeInitResult(cmd *cobra.Command, opts *options, c *collection.Collection) error {
	return opts.write(cmd.OutOrStdout(), struct {
		Path      string   `json:"path"`
		Config    string   `json:"config"`
		NextSteps []string `json:"next_steps"`
	}{c.Path, filepath.Join(c.Path, collection.ConfigFilename), []string{"means agents"}}, "Initialized collection at "+c.Path+"\nNext: run means agents to refine workflow guidance and agent discovery.\n")
}
