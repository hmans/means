package cli

import (
	"encoding/json"
	"io"

	"github.com/spf13/cobra"
)

func (o *options) write(w io.Writer, value any, text string) error {
	if o.json {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	_, err := io.WriteString(w, text)
	return err
}

func (o *options) writeHelp(cmd *cobra.Command) error {
	usage := cmd.UsageString()
	description := cmd.Long
	if description == "" {
		description = cmd.Short
	}
	return o.write(cmd.OutOrStdout(), struct {
		Command     string `json:"command"`
		Description string `json:"description"`
		Usage       string `json:"usage"`
	}{cmd.CommandPath(), description, usage}, description+"\n\n"+usage)
}
