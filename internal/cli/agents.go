package cli

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed agents.md
var agentGuide string

func newAgentsCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:     "agents [command]",
		Short:   "Learn how to set up and use means with agents",
		Long:    "Print the built-in agent guide: collection design, workflows, discovery through project instructions, skills or MCP, and routine operations. Supply a command (including nested commands) for its complete reference.\n\nNo existing collection is required. This command does not create files or install integrations. Text output is Markdown; --json returns {\"guide\":\"...\"} for the guide or {\"command\":\"...\",\"description\":\"...\",\"usage\":\"...\"} for a command reference.",
		Example: "  means help agents\n  means help agents search\n  means help agents schema set --json",
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
				return opts.writeCommandHelp(target, true)
			}
			return opts.write(cmd.OutOrStdout(), struct {
				Guide string `json:"guide"`
			}{agentGuide}, agentGuide)
		},
	}
}
