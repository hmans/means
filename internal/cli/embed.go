package cli

import (
	"fmt"

	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

func newEmbedCommand(opts *options) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Refresh collection embeddings using this machine's settings",
		Long: `Refresh missing/stale embeddings and remove obsolete vectors.
Enable embeddings in the machine-local user config and set provider credentials
first. Title/body content is sent to the provider and may incur charges.
--force discards existing vectors before rebuilding; failed runs keep completed
batches. Ordinary runs resume. Search does not backfill missing vectors.
Configure $XDG_CONFIG_HOME/means/config.yaml (default ~/.config/means/config.yaml):
  embeddings:
    provider: openai
    model: text-embedding-3-small
    dimensions: 512
Set OPENAI_API_KEY, or choose provider: openrouter with model:
openai/text-embedding-3-small and OPENROUTER_API_KEY. The corresponding large
models are also supported. Defaults are the provider's small model and 512
dimensions. Omit embeddings or set it to false to disable. Settings apply to
every collection on this machine; restart MCP after changes. Credentials belong
in the environment. MEANS_EMBEDDING_BASE_URL and MEANS_EMBEDDING_API_KEY override
the provider endpoint and key. Changing provider/model/dimensions/endpoint makes
stored vectors stale. Move legacy MEANS.md embeddings into the user config.
Enabled create/update/edit operations maintain vectors before saving documents;
API/cache failures prevent writes. Delete removes its vector without an API call.
Text reports counts and diagnostics; JSON returns documents, diagnostics, has_more.
Malformed documents are skipped with diagnostics; API/config/filesystem errors exit 1.`,
		Example: "  means embed --path ./memory --json\n  means embed --force --path ./memory\n  means search --semantic \"why did we choose Go\" --path ./memory --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			result, err := c.WithEmbeddingProgress(opts.embeddingProgress(cmd)).EmbedWithOptions(collection.EmbedOptions{Force: force})
			if err != nil {
				return err
			}
			for i := range result.Documents {
				result.Documents[i].Body = nil
			}
			text := fmt.Sprintf("Indexed %d documents.\n", len(result.Documents))
			for _, d := range result.Diagnostics {
				text += fmt.Sprintf("%s: %s\n", d.Path, d.Error)
			}
			return opts.write(cmd.OutOrStdout(), result, text)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Clear stored vectors and re-embed every valid document (fresh API charges; failures retain only completed batches)")
	return cmd
}
