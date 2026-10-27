// Package cli implements the means command-line interface.
package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hmans/means/internal/collection"
	"github.com/hmans/means/internal/userconfig"
	"github.com/spf13/cobra"
)

type options struct {
	path          string
	json          bool
	helpErr       error
	config        *userconfig.Config
	configErr     error
	configWarning error
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
	if opts.configWarning != nil && command.Name() != "mcp" {
		return opts.write(stderr, map[string]string{"warning": opts.configWarning.Error()}, "Warning: "+opts.configWarning.Error()+"\n")
	}
	return nil
}

func newRootCommand(version string) (*cobra.Command, *options) {
	opts := &options{}
	root := &cobra.Command{
		Use:              "means",
		Short:            "Structured data in Markdown files",
		Long:             overviewHelp,
		Args:             cobra.NoArgs,
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() != "version" && cmd.Name() != "help" && cmd.Name() != "means" && cmd.Name() != "agents" && !strings.HasPrefix(cmd.CommandPath(), "means help ") {
				opts.config, opts.configWarning, opts.configErr = userconfig.Open()
				if opts.configErr != nil {
					return opts.configErr
				}
			}
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
			return opts.writeBanner(cmd)
		},
	}
	root.PersistentFlags().StringVar(&opts.path, "path", ".", "Directory to work with (defaults to the current working directory)")
	root.PersistentFlags().BoolVar(&opts.json, "json", false, "Return CLI responses, help, and errors as JSON")
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		opts.helpErr = opts.writeHelp(cmd)
	})
	help := &cobra.Command{
		Use:   "help [command]",
		Short: "Show usage and command help",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				root.InitDefaultHelpFlag()
				return opts.writeHelp(root)
			}
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
	}
	help.AddCommand(&cobra.Command{
		Use: "humans", Short: "Compatibility alias for help", Hidden: true, Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root.InitDefaultHelpFlag()
			return opts.writeHelp(root)
		},
	}, newAgentsCommand(opts))
	root.SetHelpCommand(help)
	agentsAlias := newAgentsCommand(opts)
	agentsAlias.Hidden = true
	root.AddCommand(agentsAlias, newVersionCommand(opts, version), newInitCommand(opts), newSchemaCommand(opts), newEmbedCommand(opts),
		newCreateCommand(opts), newReadCommand(opts), newShowCommand(opts), newListCommand(opts), newSearchCommand(opts), newUpdateCommand(opts), newDeleteCommand(opts), newDoctorCommand(opts), newMCPCommand(opts, version), newInfoCommand(opts), newEditCommand(opts))
	root.AddGroup(&cobra.Group{ID: "documents", Title: "Documents:"}, &cobra.Group{ID: "collection", Title: "Collection:"}, &cobra.Group{ID: "other", Title: "Help and integrations:"})
	root.SetHelpCommandGroupID("other")
	for _, cmd := range root.Commands() {
		switch cmd.Name() {
		case "create", "list", "search", "show", "read", "edit", "update", "delete":
			cmd.GroupID = "documents"
		case "init", "info", "schema", "doctor", "embed":
			cmd.GroupID = "collection"
		default:
			cmd.GroupID = "other"
		}
	}
	return root, opts
}

func (opts *options) loadCollection() (*collection.Collection, error) {
	c, err := collection.Load(opts.path)
	if err != nil {
		return nil, err
	}
	return c.WithEmbeddings(opts.config.Embeddings), nil
}
