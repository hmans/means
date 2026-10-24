package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/hmans/means/internal/collection"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

func newUpdateCommand(opts *options) *cobra.Command {
	var input string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Patch a document only if its content hash still matches",
		Long: `Patch an existing document in --path (default: current directory).

Read first:
  IDs are local to the selected collection. Lookup stops at subdirectories
  containing MEANS.md; select a nested collection with its own --path.
  Run "means read <id> --json" and keep its hash. This is SHA-256 of the exact
  stored Markdown bytes, including frontmatter, whitespace, and line endings.
  list omits hashes. A missing or stale hash cannot update a document.

Input:
  --input FILE reads a JSON file relative to the current working directory;
  --input - reads stdin until EOF. Supply one object with required hash and
  at least one of frontmatter or body. Unknown fields are rejected.
  {"hash":"<64 lowercase hex characters>","frontmatter":{"data":{"status":"done"}}}
  frontmatter may patch title and/or data. id is immutable: any supplied id,
  including null or the current value, is rejected. Other top-level fields are
  reserved. title must remain a nonempty string and data must remain an object.
  frontmatter uses JSON Merge Patch (RFC 7396): omitted
  keys stay unchanged, objects merge recursively, null deletes a key, and arrays
  replace as a whole. Null cannot set a literal null field using this patch format.
  body is an optional replacement string: omit to retain it, use "" to clear it.
  body:null and frontmatter:null are rejected. An empty frontmatter object is
  allowed as a no-op patch; the hash and resulting schema are still checked.

Effects:
  Preserve the ID and filename, even when the title changes. Check the hash
  and validate the means envelope and resulting data against MEANS.md
  before replacing the file. Conflicts and validation errors leave it unchanged.
  On conflict, read again, review the new content, and recompute your patch.
  Body-only changes preserve the original frontmatter text. Metadata changes
  re-encode YAML and can change formatting/comments. An unchanged result retains
  the original bytes and hash. File permission bits are preserved.
  Updates recheck the hash before replacement and write a same-directory temporary
  file, then rename it over the original (atomic on supported filesystems).
  There are no locks: a simultaneous write between the final check and rename
  can still be overwritten.

Output:
  Text: the ID followed by a newline.
  --json: {"id":"...","hash":"...","frontmatter":{...},"body":"..."}.
  Errors: stderr, exit 1; --json returns {"error":"..."}. There is no force option.
  Success returns the new hash. Retrying with the old hash after a content change
  fails; read again to find out whether a previous attempt succeeded.`,
		Example: `  means read <id> --path ./memory --json
  means update <id> --path ./memory --input - --json <<'JSON'
  {
    "hash": "<hash from read>",
    "frontmatter": {"title": "Corrected title", "data": {"tags": ["decision"]}},
    "body": "Updated Markdown body.\n"
  }
JSON`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" {
				return fmt.Errorf("--input is required; supply a file path or - for stdin")
			}
			var data []byte
			var err error
			if input == "-" {
				data, err = io.ReadAll(cmd.InOrStdin())
			} else {
				data, err = os.ReadFile(input)
			}
			if err != nil {
				return fmt.Errorf("read update input: %w", err)
			}
			hash, patch, err := parseUpdateInput(data)
			if err != nil {
				return err
			}
			c, err := collection.Load(opts.path)
			if err != nil {
				return err
			}
			doc, err := c.Update(args[0], hash, patch)
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), doc, doc.ID+"\n")
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "Update JSON with hash and patch fields from FILE, or - for stdin (required)")
	return cmd
}

func parseUpdateInput(data []byte) (string, collection.Patch, error) {
	var patch collection.Patch
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return "", patch, fmt.Errorf("parse update input JSON: %w", err)
	}
	envelope, ok := value.(map[string]any)
	if !ok {
		return "", patch, fmt.Errorf("update input must be an object")
	}
	for key := range envelope {
		if key != "hash" && key != "frontmatter" && key != "body" {
			return "", patch, fmt.Errorf("unknown update field %q", key)
		}
	}
	hash, ok := envelope["hash"].(string)
	if !ok {
		return "", patch, fmt.Errorf("hash is required; read the document first")
	}
	if value, exists := envelope["frontmatter"]; exists {
		patch.Frontmatter, ok = value.(map[string]any)
		if !ok {
			return "", patch, fmt.Errorf("frontmatter patch must be an object")
		}
	}
	if value, exists := envelope["body"]; exists {
		body, ok := value.(string)
		if !ok {
			return "", patch, fmt.Errorf("body must be a string")
		}
		patch.Body = &body
	}
	return hash, patch, nil
}
