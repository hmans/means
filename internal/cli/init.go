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
		Long: `Create a collection in --path (default: current directory).
Use the example schema, --schema FILE|- for JSON Schema, or --input FILE|- for
a complete MEANS.md. The two input flags are exclusive. Existing MEANS.md files
are never overwritten. JSON returns path, config (the MEANS.md path), and next_steps.
Invalid input exits 1 without creating collection files. Schema defaults to
Draft 2020-12; external references are not supported.

Choose a schema for your application data, then write the collection workflow
in the Markdown body of MEANS.md. Run info to inspect it and doctor to check
documents. For agent use, make the collection discoverable through project
instructions, a skill, or MCP (see "means help mcp"). Initializing storage alone
does not establish that workflow or discovery path. Read "means help agents" for
collection structure, schema design, and MEANS.md authoring.`,
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
	}{c.Path, filepath.Join(c.Path, collection.ConfigFilename), []string{"means help create"}}, "Initialized collection at "+c.Path+"\nNext: run means help create to add a document.\n")
}
