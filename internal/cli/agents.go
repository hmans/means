package cli

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed agents.md
var agentsGuide string

func newAgentsCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "agents [command]",
		Short: "Learn collection structure and MEANS.md authoring",
		Long:  strings.TrimSpace(agentsGuide),
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
			cmd.InitDefaultHelpFlag()
			return opts.writeHelp(cmd)
		},
	}
}
