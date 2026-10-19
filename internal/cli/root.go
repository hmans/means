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
func Execute(args []string, stdout, stderr io.Writer, version string) error {
	root, opts := newRootCommand(version)
	if args == nil {
		args = []string{}
	}
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	err := root.Execute()
	err = errors.Join(err, opts.helpErr)
	if err != nil {
		// Flag parsing can stop before reaching --json. Re-read global flags
		// without rejecting unknown flags so error formatting is order-independent.
		root.FParseErrWhitelist.UnknownFlags = true
		_ = root.ParseFlags(args)
		writeErr := opts.write(stderr, map[string]string{"error": err.Error()}, "Error: "+err.Error()+"\n")
		return errors.Join(err, writeErr)
	}
	return nil
}

func newRootCommand(version string) (*cobra.Command, *options) {
	opts := &options{}
	root := &cobra.Command{
		Use:              "means",
		Short:            "Agentic memory built from Markdown files",
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
	root.PersistentFlags().BoolVar(&opts.json, "json", false, "Return responses as JSON")
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
	root.AddCommand(newVersionCommand(opts, version))
	return root, opts
}
