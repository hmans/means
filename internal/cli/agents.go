package cli

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed agents.md
var agentGuide string

func newAgentsCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:     "agents",
		Short:   "Learn how to set up and use means with agents",
		Long:    "Print the built-in agent guide: collection design, workflows, discovery through project instructions, skills or MCP, and routine operations.\n\nNo existing collection is required. This command does not create files or install integrations. Text output is Markdown; --json returns {\"guide\":\"...\"} containing the same guide.",
		Example: "  means agents\n  means agents --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.write(cmd.OutOrStdout(), struct {
				Guide string `json:"guide"`
			}{agentGuide}, agentGuide)
		},
	}
}
