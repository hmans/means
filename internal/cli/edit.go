package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

func newEditCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use: "edit <id>", Short: "Edit a document with your text editor",
		Long: `Open a private temporary Markdown copy in $VISUAL, falling back to $EDITOR.
Set the variable to an executable name or path; editor arguments are not supported.
Use an editor wrapper script when arguments are needed. The editor must wait until
editing is complete. Editor output goes to stderr, including in --json mode.

Keep the document ID unchanged. Title and data replace the original metadata;
removed keys are removed and literal null values are retained. Validate the
complete document against the current schema before saving. Metadata edits may
reformat YAML; body-only edits preserve original frontmatter formatting.
The original content hash protects against observed concurrent edits; no force
or automatic retry. As with update, an external write can still race replacement.
Embedding API/cache failures prevent saving. On any editor or save failure,
retain the temporary copy and report its path for recovery. On success, remove it.
Text returns the ID; JSON returns the document and new hash, as update does.`,
		Example: "  EDITOR=vi means edit abc12\n  means edit abc12 --path ./issues",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			editor := strings.TrimSpace(os.Getenv("VISUAL"))
			if editor == "" {
				editor = strings.TrimSpace(os.Getenv("EDITOR"))
			}
			if editor == "" {
				return fmt.Errorf("set VISUAL or EDITOR to an editor executable that waits until editing is complete")
			}
			executable, err := exec.LookPath(editor)
			if err != nil {
				return fmt.Errorf("find editor %q (use a wrapper script for arguments): %w", editor, err)
			}
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			doc, err := c.Read(args[0])
			if err != nil {
				return err
			}
			file, err := os.CreateTemp("", "means-edit-*.md")
			if err != nil {
				return err
			}
			name := file.Name()
			if _, err := file.WriteString(doc.Markdown); err != nil {
				file.Close()
				os.Remove(name)
				return err
			}
			if err := file.Close(); err != nil {
				os.Remove(name)
				return err
			}
			recoverError := func(err error) error { return fmt.Errorf("%w; edited copy retained at %s", err, name) }
			process := exec.CommandContext(cmd.Context(), executable, name)
			process.Stdin, process.Stdout, process.Stderr = cmd.InOrStdin(), cmd.ErrOrStderr(), cmd.ErrOrStderr()
			if err := process.Run(); err != nil {
				return recoverError(fmt.Errorf("editor failed: %w", err))
			}
			content, err := os.ReadFile(name)
			if err != nil {
				return recoverError(err)
			}
			// Reload the schema after the editor closes, before validating and saving.
			c, err = collection.Load(opts.path)
			if err != nil {
				return recoverError(err)
			}
			updated, err := c.WithEmbeddings(opts.config.Embeddings).WithEmbeddingProgress(opts.embeddingProgress(cmd)).ReplaceMarkdown(doc.ID, doc.Hash, content)
			if err != nil {
				return recoverError(err)
			}
			if err := os.Remove(name); err != nil {
				return fmt.Errorf("saved document %s, but could not remove edited copy %s: %w", doc.ID, name, err)
			}
			return opts.write(cmd.OutOrStdout(), updated, updated.ID+"\n")
		},
	}
}
