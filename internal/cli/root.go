// Package cli implements the means command-line interface.
package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
)

type options struct {
	path    string
	json    bool
	helpErr error
}

// Execute runs one invocation, writing responses to stdout and errors to stderr.
// The caller is responsible for translating the returned error into an exit code.
func Execute(args []string, stdin io.Reader, stdout, stderr io.Writer, version string) error {
	root, opts := newRootCommand(version)
	if args == nil {
		args = []string{}
	}
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	command, err := root.ExecuteC()
	err = errors.Join(err, opts.helpErr)
	if err != nil {
		if errors.Is(err, ErrUnhealthy) || errors.Is(err, ErrSchemaRejected) {
			return err
		}
		// Flag parsing can stop before reaching --json. Re-read command flags
		// without rejecting unknown flags so error formatting is order-independent.
		command.FParseErrWhitelist.UnknownFlags = true
		_ = command.ParseFlags(args)
		writeErr := opts.write(stderr, map[string]string{"error": err.Error()}, "Error: "+err.Error()+"\n")
		return errors.Join(err, writeErr)
	}
	return nil
}

func newRootCommand(version string) (*cobra.Command, *options) {
	opts := &options{}
	root := &cobra.Command{
		Use:   "means",
		Short: "Agentic memory built from Markdown files",
		Long: `means provides typed Markdown collections for agents and applications.

Setting up means for agents? Run "means agents" for the complete onboarding guide:
design a schema, write MEANS.md, and make the collection discoverable through
project instructions, a skill, or MCP. Creating storage alone does not finish agent setup.

A collection is a directory containing MEANS.md with YAML configuration (title,
description, and a JSON Schema under schema) and Markdown workflow guidance. The files are authoritative and can be edited
directly. Means owns frontmatter.id (generated and immutable) and frontmatter.title
(required); application fields live under frontmatter.data. Filenames are storage
details. Read and update use the stored ID, even after file renames or moves.

Start with "means init --help" for the complete schema-authoring workflow,
file and stdin input examples, validation rules, and response formats.
Then use "means create --help" to store a validated document, "means list" to
discover IDs and titles, and "means show <id>" to view rendered Markdown.
Use "means read <id>" to retrieve the exact stored Markdown.
Use "means mcp --help" to expose a collection to an MCP client over stdio.
Use "means update --help" to patch a document using the hash from read --json.
Use "means delete --help" to permanently remove a document using its content hash.
Use "means schema set --help" to validate and install a replacement schema.
Use "means doctor" to check all documents
against the current schema, or "means search" to find documents by text/data.
Use "means embed --help" for opt-in OpenAI embeddings, automatic maintenance,
and semantic search with "means search --semantic".

Every command accepts --path and --json before or after the command name.
--path defaults to the current working directory; relative paths resolve there.
Means works with that exact directory and does not search parent directories.
--json returns two-space-indented JSON, including help and errors.
The mcp command always uses MCP message framing and structured JSON tool results;
--json only affects its CLI help and startup errors.
Successful responses go to stdout (exit 0); errors go to stderr (exit 1) as
{"error":"..."} in JSON mode. Doctor reports invalid documents on stdout and
exits 1. Schema set rejects incompatible documents with a diagnostic report on
stderr (exit 1), including an error field in JSON mode; see "means schema set --help".
Parse the entire JSON response, not individual lines.
Use "means help COMMAND" or "means COMMAND --help" to discover each command.`,
		Args:             cobra.NoArgs,
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.path == "" {
				return fmt.Errorf("--path must not be empty")
			}
			path, err := filepath.Abs(opts.path)
			if err != nil {
				return fmt.Errorf("resolve --path: %w", err)
			}
			opts.path = path
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.writeHelp(cmd)
		},
	}
	root.PersistentFlags().StringVar(&opts.path, "path", ".", "Directory to work with (defaults to the current working directory)")
	root.PersistentFlags().BoolVar(&opts.json, "json", false, "Return CLI responses, help, and errors as JSON")
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		opts.helpErr = opts.writeHelp(cmd)
	})
	root.SetHelpCommand(&cobra.Command{
		Use:   "help [command]",
		Short: "Show help for a command",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, remaining, err := root.Find(args)
			if err != nil {
				return err
			}
			if len(remaining) != 0 {
				return fmt.Errorf("unknown help topic %q", remaining[0])
			}
			target.InitDefaultHelpFlag()
			return opts.writeHelp(target)
		},
	})
	root.AddCommand(newAgentsCommand(opts), newVersionCommand(opts, version), newInitCommand(opts), newSchemaCommand(opts), newEmbedCommand(opts),
		newCreateCommand(opts), newReadCommand(opts), newShowCommand(opts), newListCommand(opts), newSearchCommand(opts), newUpdateCommand(opts), newDeleteCommand(opts), newDoctorCommand(opts), newMCPCommand(opts, version))
	return root, opts
}
