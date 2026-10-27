package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

const humanHelp = `means manages collections of Markdown documents with structured data and schemas.

Get started:
  means init --path ./memory      Create a collection
  means list --path ./memory      List its documents
  means show <id> --path ./memory  Read a document

Use --path to select a collection (default: the current directory).
Use "means <command> --help" for command details and examples.
Use "means help agents" for agent setup and automation guidance.`

func (o *options) writeBanner(cmd *cobra.Command) error {
	type entry struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Group       string `json:"group"`
	}
	commands := []entry{}
	var text strings.Builder
	fmt.Fprintf(&text, "means — %s\n\nRun \"means help\" for human guidance.\nRun \"means help agents\" for agent guidance.\n\nAvailable commands:\n", cmd.Short)
	for _, group := range cmd.Groups() {
		fmt.Fprintf(&text, "\n%s\n", group.Title)
		for _, child := range cmd.Commands() {
			if child.GroupID != group.ID {
				continue
			}
			if child.Hidden || (!child.IsAvailableCommand() && child.Name() != "help") {
				continue
			}
			commands = append(commands, entry{child.Name(), child.Short, group.ID})
			fmt.Fprintf(&text, "  %-10s %s\n", child.Name(), child.Short)
		}
	}
	return o.write(cmd.OutOrStdout(), struct {
		Command     string   `json:"command"`
		Description string   `json:"description"`
		Commands    []entry  `json:"commands"`
		NextSteps   []string `json:"next_steps"`
	}{cmd.Name(), cmd.Short, commands, []string{"means help", "means help agents"}}, text.String())
}

func embeddingProgress(cmd *cobra.Command) func(string) {
	logger := log.New(cmd.ErrOrStderr(), "means: ", 0)
	return func(message string) { logger.Print(message) }
}

func (o *options) embeddingProgress(cmd *cobra.Command) func(string) {
	if o.json {
		return nil
	}
	return embeddingProgress(cmd)
}

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
	return o.writeCommandHelp(cmd, o.json)
}

func (o *options) writeCommandHelp(cmd *cobra.Command, detailed bool) error {
	if !detailed {
		if example, ok := humanCommandExamples[cmd.CommandPath()]; ok {
			original := cmd.Example
			defer func() { cmd.Example = original }()
			cmd.Example = example
		}
	}
	usage := cmd.UsageString()
	description := cmd.Long
	if description == "" {
		description = cmd.Short
	}
	if !detailed {
		if concise, ok := humanCommandHelp[cmd.CommandPath()]; ok {
			description = concise + "\n\nFull reference: means help agents " + strings.TrimPrefix(cmd.CommandPath(), "means ")
		}
	}
	return o.write(cmd.OutOrStdout(), struct {
		Command     string `json:"command"`
		Description string `json:"description"`
		Usage       string `json:"usage"`
	}{cmd.CommandPath(), description, usage}, description+"\n\n"+usage)
}
