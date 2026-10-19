package cli

import "github.com/spf13/cobra"

func newVersionCommand(opts *options, version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the means version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.write(cmd.OutOrStdout(), struct {
				Version string `json:"version"`
			}{version}, "means "+version+"\n")
		},
	}
}
