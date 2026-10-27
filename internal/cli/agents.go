package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Keep old audience-specific routes as aliases for the canonical help.
func newAgentsCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:    "agents [command]",
		Short:  "Compatibility alias for help",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				target, remaining, err := cmd.Root().Find(args)
				if err != nil {
					return err
				}
				if len(remaining) != 0 {
					return fmt.Errorf("unknown help topic %q", remaining[0])
				}
				target.InitDefaultHelpFlag()
				return opts.writeHelp(target)
			}
			cmd.Root().InitDefaultHelpFlag()
			return opts.writeHelp(cmd.Root())
		},
	}
}
