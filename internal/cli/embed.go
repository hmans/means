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
		Long: `Refresh missing or stale document embeddings and remove obsolete entries.

Use --force to clear all stored vectors before regenerating every valid document,
including unchanged documents. Markdown is preserved. If regeneration fails,
only batches completed during this run remain cached. Run embed without --force
to resume without discarding that progress. This incurs fresh API charges.

Opt in by manually editing the Means user config:
  $XDG_CONFIG_HOME/means/config.yaml (default ~/.config/means/config.yaml)
On first CLI invocation or MCP startup, Means creates a commented example if the
file is missing; embeddings stay disabled. Existing files are never overwritten.
Set the following in that file (plain YAML, without Markdown delimiters):
  embeddings:
    provider: openrouter
Settings apply to every collection used on this machine. CLI reads settings each
invocation; MCP reads them at startup, so restart it after editing the config.
Bootstrap failures leave embeddings disabled and warn on successful commands;
existing malformed/unreadable configs fail rather than silently disabling them.
Provider is required: openai or openrouter. Omit embeddings or set it to false
to disable. Model defaults to text-embedding-3-small for openai and
openai/text-embedding-3-small for openrouter. Both also support the corresponding
text-embedding-3-large model. OpenRouter requires the openai/ model prefix;
direct OpenAI rejects it. Optional dimensions must be positive and at most
1536 (small) or 3072 (large); omitted dimensions default to 512.

Set OPENAI_API_KEY for direct OpenAI access, or OPENROUTER_API_KEY for OpenRouter.
No base URL is required: openai uses https://api.openai.com/v1; openrouter uses
https://openrouter.ai/api/v1. MEANS_EMBEDDING_BASE_URL overrides the endpoint for
proxies or testing. MEANS_EMBEDDING_API_KEY overrides the provider-specific key.
Endpoint overrides do not change provider, model, or credential selection.
Missing credentials fail; there is no fallback to another provider's key.
Endpoint overrides and keys are never read from collection documents. Changing
provider, model, dimensions, or endpoint invalidates the local vector profile.

Migration: move any embeddings block from MEANS.md into the user config and
remove it from MEANS.md, including embeddings: false. Legacy collection settings
are rejected with migration guidance. There are no per-collection overrides and
no automatic configuration migration. Collections keep only identity/schema/workflow.
Compatible existing local vector caches remain reusable. Different host profiles
invalidate those vectors. Run embed on each configured system to backfill documents.

Opt-in enables automatic maintenance on create/update/delete in CLI and MCP for
all collections, sending changed title/body input to the configured provider.
Create/update store the embedding before changing the document; API or cache
write failures leave the document unchanged. Delete removes its vector locally
before removing the file and needs no API call. These two files are not one
transaction: a failed document write or crash can leave stale cache data.
Run embed to reconcile it against current files. Semantic search embeds only the
query and uses current stored vectors; it skips missing or stale vectors without
refreshing or pruning the cache.

Each vector represents "Title: " + title + two newlines + the exact Markdown body.
Application data is excluded. Data-only edits reuse vectors. Inputs are not
truncated or chunked: over-limit inputs fail at the API. Requests have a 60-second
timeout and are not automatically retried. Generating vectors sends content to
the configured endpoint and may incur charges.

The local .means/embeddings.jsonl cache is separate from parsed-document metadata.
It stores float vectors, input hashes, a format/model/dimension/endpoint profile,
and a checksum. Cache replacements are atomic; an embeddings.lock serializes
embedding work with a 5-second lock wait; semantic search reads without it.
Missing/corrupt/obsolete caches regenerate on demand and may incur charges.
Filesystem errors and symlinked cache paths fail. No database, daemon, watcher,
or frontmatter changes are required.

Run this command after enabling embeddings for existing documents or after direct
edits, deletes, and changes to nested collection boundaries. Semantic search uses
stored vectors and does not refresh them.
Read/list/full-text/literal searches do not generate embeddings. Results reflect
files observed during scanning, not an atomic snapshot of external edits.

Output: text reports the number of indexed documents and diagnostics; --json returns
{"documents":[{"id":"...","frontmatter":{...}}],"diagnostics":[],"has_more":false}.
Vectors and document bodies are omitted. Malformed/duplicate documents are excluded
with diagnostics (exit 0); configuration/API/filesystem errors use stderr and exit 1.
Completed embeddings are retained if a later document fails during a refresh.
Refreshes batch up to 32 changed documents per API request, process batches
sequentially, and replace the cache once per completed batch. A failed or invalid
batch is not applied; earlier completed batches are retained for the next run.
Provider input limits still apply; an over-limit document fails its whole batch.
Progress goes to stderr unless --json is set: document IDs before API calls
for single writes, completed/total documents for batches, and counts of generated,
reused, and removed vectors after refresh. It does not
include document text, search query text, vectors, or credentials. Stdout remains
the command result. Redirect stderr to suppress progress.`,
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
