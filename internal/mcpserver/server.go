// Package mcpserver exposes collection operations through MCP.
package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/hmans/means/internal/collection"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const instructions = `This server works with one Markdown collection. Call info
to learn the schema for frontmatter.data before creating documents. Means owns
frontmatter.id (generated and immutable) and frontmatter.title (required nonempty
string). Creation takes title and optional data (default {}); do not supply id.
List or search documents to find IDs, then read by ID for full bodies. Search
matches all query tokens in any order across title/body, ranks with a title boost,
and filters frontmatter.data by value. Set literal:true for substring matching.
With embeddings configured in MEANS.md, create/update/delete maintain local
title/body embeddings automatically. API/cache errors prevent document mutations.
Set semantic:true on search for cosine ranking of stored document vectors; this
sends only the query to the configured embedding endpoint. Search never changes
document vectors and skips missing or stale vectors; use the CLI command
means embed to refresh them. It
requires a nonempty query and cannot combine with literal:true. Semantic hits
have score instead of match; min_score defaults to 0.3 (inclusive, before limit).
Override min_score from -1 to 1; -1 includes all scores. It requires semantic:true.
Embedding refresh is CLI-only, via means embed (optionally --force).
Embeddings use a separate
.means/embeddings.jsonl cache and lock. No watcher is used. API calls may incur
charges. Keys and endpoint overrides come from the server process environment.
Full-text search refreshes a disposable .means/search/ index on each call, reads
and hashes all visible documents, and closes the index before returning. There
is no watcher. Concurrent searches wait up to five seconds for a cache lock.
IDs are stored values, independent of filenames. Lookup scans visible
subdirectories, stopping at any directory containing MEANS.md, even if invalid.
Nested collections have independent IDs, schemas, and MEANS.md guidance.
Renames within a collection preserve identity. Duplicate IDs are errors. Patches
may change title and data, never id.
Title changes preserve existing filenames.
Creation validates the current schema and is not idempotent: do not retry a
successful creation. Reads and lists validate the means envelope but not the data schema;
inspect diagnostics in list results. Schema and document edits take effect
on the next tool call. Read before update: supply its hash and only the fields to
patch. A stale hash fails; read again and review changes before retrying. Metadata
uses JSON Merge Patch (objects merge, null deletes, arrays replace); body replaces
the whole body when supplied. Delete permanently removes a document by ID and
hash from read, without validating existing application data. Missing or duplicate
IDs, malformed envelopes, and stale hashes fail. There is no undo. Directories
and configuration remain. Concurrent writes can race the final removal.
Use schema_set with a schema object or boolean and optional force boolean to
validate and install a replacement. Document violations block installation unless
force is true; diagnostics are always returned. Invalid schemas and filesystem
errors cannot be forced. No documents are rewritten. Checks cover current data,
not all possible future data, and concurrent edits can race installation.
The Markdown body of MEANS.md supplies collection-specific
workflow guidance. It is not a document and is excluded from listing and reading.
Call info for current guidance; restart the server to refresh discovery
descriptions and connection instructions after editing collection metadata.`

const startupInstructions = `Call info for the current schema and detailed guidance.
List or search to find IDs, then read before updating or deleting with its hash.
Deletion is permanent; a repeated deletion fails. Create and
update validate frontmatter.data against the current schema. Stored IDs are not
filenames. Creation is not idempotent; do not retry a successful create.
Use schema_set to validate and install schema changes; force permits document violations.`

func collectionPurpose(info *collection.Description) string {
	purpose := "Collection: " + info.Title + "."
	if info.Description != "" {
		purpose += "\n" + info.Description
	}
	return purpose
}

func collectionInstructions(info *collection.Description) string {
	text := collectionPurpose(info) + "\n\n" + instructions
	if info.Instructions != "" {
		text += "\n\nCollection workflow (MEANS.md):\n" + info.Instructions
	}
	return text
}

func collectionStartupInstructions(info *collection.Description) string {
	text := collectionPurpose(info) + "\n\n" + startupInstructions
	if info.Instructions != "" {
		text += "\n\nCollection workflow (MEANS.md):\n" + info.Instructions
	}
	return text
}

// New validates the selected collection and builds a server bound to its path.
// Each call reloads the collection so external schema edits take effect immediately.
func New(path, version string) (*mcp.Server, error) {
	return NewWithEmbeddingProgress(path, version, nil)
}

// NewWithEmbeddingProgress reports embedding work through progress. Concurrent
// calls may invoke progress concurrently; callers must serialize their output.
func NewWithEmbeddingProgress(path, version string, progress func(string)) (*mcp.Server, error) {
	c, err := collection.Load(path)
	if err != nil {
		return nil, err
	}
	path = c.Path
	info := c.Describe()
	server := mcp.NewServer(&mcp.Implementation{Name: "means", Title: info.Title, Description: info.Description, Version: version},
		&mcp.ServerOptions{Instructions: collectionStartupInstructions(info)})
	for _, tool := range []toolDefinition{
		{
			name:        "schema_set",
			description: "Validate and install a JSON Schema for this collection. Pass schema as an object or boolean, not a string. Optional force (default false) permits existing document violations but never invalid schemas or filesystem errors. Returns installed, healthy, checked, and diagnostics. Rejection is a tool error with structured diagnostics and leaves the schema unchanged. Checks current data, not abstract schema compatibility. Does not rewrite documents; concurrent edits can race installation. Can repair an invalid or missing schema in otherwise valid MEANS.md frontmatter while this server remains running. Malformed YAML requires a direct edit. Preserves identity and Markdown body; frontmatter may be reformatted. Restart to refresh discovery metadata.",
			schema:      `{"type":"object","properties":{"schema":{"type":["object","boolean"],"description":"Complete self-contained JSON Schema. Draft 2020-12 by default; external references are unsupported."},"force":{"type":"boolean","default":false}},"required":["schema"],"additionalProperties":false}`,
			destructive: true,
			runPath: func(path string, arguments json.RawMessage) (any, error) {
				// Keep the raw schema bytes so the supplied key order survives.
				var args struct {
					Schema json.RawMessage `json:"schema"`
					Force  bool            `json:"force"`
				}
				if err := json.Unmarshal(arguments, &args); err != nil {
					return nil, err
				}
				return collection.SetSchema(path, args.Schema, args.Force)
			},
		},
		{
			name:        "info",
			description: "Return current collection identity, schema, path, and full guidance. Call before creating documents.",
			schema:      `{"type":"object","additionalProperties":false}`,
			readOnly:    true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				current := c.Describe()
				return map[string]any{"path": c.Path, "schema": c.Schema,
					"title": current.Title, "description": current.Description,
					"embeddings":   c.Embeddings(),
					"instructions": collectionInstructions(current)}, nil
			},
		},
		{
			name:        "list",
			description: "List valid document IDs and frontmatter in filename order, without bodies or hashes. Returns documents, diagnostics, and has_more. limit counts valid documents (default 20; 0 returns all). Malformed documents and every duplicate occurrence are excluded and reported in diagnostics regardless of the limit. Read an ID before updating it.",
			schema:      `{"type":"object","properties":{"limit":{"type":"integer","minimum":0,"description":"Maximum documents; default 20, 0 for all."}},"additionalProperties":false}`,
			readOnly:    true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				limit, err := limitArg(args)
				if err != nil {
					return nil, err
				}
				return c.List(limit)
			},
		},
		{
			name:        "search",
			description: "Find documents by ranked full text and/or a partial data filter. All lowercase Unicode query tokens must match across title/body; no stemming, stop-word removal, or query syntax. Title matches are boosted; score ties sort by ID. literal:true uses case-insensitive substring matching in filename order. No query uses filename order. Full-text queries refresh .means/search/ from file hashes, then close it; no daemon. Returns IDs, frontmatter, excerpts, and diagnostics, without bodies or hashes. Filters apply before limit (default 20, 0 for all).",
			schema:      `{"type":"object","properties":{"query":{"type":"string","description":"Text query. Full text requires all terms; semantic search ranks similarity."},"min_score":{"type":"number","minimum":-1,"maximum":1,"default":0.3,"description":"Inclusive cosine similarity threshold, applied before limit. Requires semantic:true. Use -1 for all scores."},"semantic":{"type":"boolean","default":false,"description":"Opt-in semantic search: embed only the query, rank stored document vectors, return cosine scores instead of excerpts. Never changes document vectors; skips missing or stale vectors (refresh with CLI means embed). Requires a nonempty query and API credentials; cannot combine with literal. May incur charges."},"literal":{"type":"boolean","default":false,"description":"Use the original literal substring search without an index."},"filter":{"type":"object","description":"Partial match against frontmatter.data; e.g. {\"priority\":\"normal\",\"type\":\"feature\"}."},"limit":{"type":"integer","minimum":0,"description":"Maximum matches; default 20, 0 for all."}},"additionalProperties":false}`,
			readOnly:    true,
			openWorld:   true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				limit, err := limitArg(args)
				if err != nil {
					return nil, err
				}
				query, _ := args["query"].(string)
				filter, _ := args["filter"].(map[string]any)
				literal, _ := args["literal"].(bool)
				semantic, _ := args["semantic"].(bool)
				var minScore *float64
				if value, exists := args["min_score"]; exists {
					n, err := value.(json.Number).Float64()
					if err != nil {
						return nil, fmt.Errorf("min_score must be a finite number from -1 to 1")
					}
					minScore = &n
				}
				return c.Search(collection.SearchOptions{Query: query, Filter: filter, Limit: limit, Literal: literal, Semantic: semantic, MinScore: minScore})
			},
		},
		{
			name:        "read",
			description: "Read by stored ID, independent of filename. Returns frontmatter, exact body, and the content hash required for update. Missing and duplicate IDs fail.",
			schema:      `{"type":"object","properties":{"id":{"type":"string","pattern":"^[A-Za-z0-9][A-Za-z0-9_-]*$","maxLength":128,"description":"Stored frontmatter.id, not a filename or path."}},"required":["id"],"additionalProperties":false}`,
			readOnly:    true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				return c.Read(args["id"].(string))
			},
		},
		{
			name:        "create",
			openWorld:   true,
			description: "Create a document with a required title, optional application data, and optional Markdown body. Call info for the data schema. Means generates the ID. Each successful call creates another document.",
			schema:      `{"type":"object","properties":{"frontmatter":{"type":"object","properties":{"title":{"type":"string","minLength":1},"data":{"type":"object","description":"Application fields validated against the schema returned by info. Omit for {}."}},"required":["title"],"additionalProperties":false},"body":{"type":"string","description":"Markdown body, preserved exactly; defaults to empty."}},"required":["frontmatter"],"additionalProperties":false}`,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				body, _ := args["body"].(string)
				return c.Create(args["frontmatter"].(map[string]any), body)
			},
		},
		{
			name:        "update",
			openWorld:   true,
			description: "Patch title/data or replace the body using the hash from read. Omit unchanged fields; null deletes nested data keys. The ID cannot change. A stale hash fails; read again before retrying.",
			schema:      `{"type":"object","properties":{"id":{"type":"string","pattern":"^[A-Za-z0-9][A-Za-z0-9_-]*$","maxLength":128},"hash":{"type":"string","pattern":"^[0-9a-f]{64}$","description":"Content hash returned by read."},"frontmatter":{"type":"object","properties":{"title":{"type":"string","minLength":1},"data":{"type":"object","description":"Merge patch for application data. Null values delete nested keys; arrays replace."}},"additionalProperties":false},"body":{"type":"string","description":"Replacement Markdown body; omit to retain."}},"required":["id","hash"],"minProperties":3,"additionalProperties":false}`,
			destructive: true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				patch := collection.Patch{}
				patch.Frontmatter, _ = args["frontmatter"].(map[string]any)
				if value, exists := args["body"]; exists {
					body := value.(string)
					patch.Body = &body
				}
				return c.Update(args["id"].(string), args["hash"].(string), patch)
			},
		},
		{
			name:        "delete",
			description: "Permanently remove one document by ID and the hash from read. Returns the deleted ID. Missing or duplicate IDs, malformed envelopes, and stale hashes fail. Does not validate existing application data. Preserves directories and configuration. No undo; concurrent writes can race removal.",
			schema:      `{"type":"object","properties":{"id":{"type":"string","pattern":"^[A-Za-z0-9][A-Za-z0-9_-]*$","maxLength":128},"hash":{"type":"string","pattern":"^[0-9a-f]{64}$","description":"Content hash returned by read."}},"required":["id","hash"],"additionalProperties":false}`,
			destructive: true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				return c.Delete(args["id"].(string), args["hash"].(string))
			},
		},
	} {
		tool.description = collectionPurpose(info) + "\n\n" + tool.description
		if err := addTool(server, path, tool, progress); err != nil {
			return nil, err
		}
	}
	return server, nil
}

func limitArg(args map[string]any) (int, error) {
	if value, exists := args["limit"]; exists {
		n, ok := new(big.Rat).SetString(value.(json.Number).String())
		if !ok || !n.IsInt() || !n.Num().IsInt64() {
			return 0, fmt.Errorf("limit is too large")
		}
		limit := int(n.Num().Int64())
		if int64(limit) != n.Num().Int64() {
			return 0, fmt.Errorf("limit is too large")
		}
		return limit, nil
	}
	return 20, nil
}

type toolDefinition struct {
	name, description, schema string
	readOnly                  bool
	openWorld                 bool
	destructive               bool
	run                       func(*collection.Collection, map[string]any) (any, error)
	runPath                   func(string, json.RawMessage) (any, error) // Operations that need no valid current schema.
}

func addTool(server *mcp.Server, path string, tool toolDefinition, progress func(string)) error {
	schema, err := jsonschema.UnmarshalJSON(bytes.NewBufferString(tool.schema))
	if err != nil {
		return err
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("tool.json", schema); err != nil {
		return err
	}
	validator, err := compiler.Compile("tool.json")
	if err != nil {
		return err
	}
	server.AddTool(&mcp.Tool{
		Name: tool.name, Description: tool.description, InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: tool.readOnly,
			DestructiveHint: &tool.destructive, OpenWorldHint: &tool.openWorld},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Decode and validate ourselves: generic SDK handlers decode arbitrary
		// metadata through float64, which can lose large integers and decimals.
		data := req.Params.Arguments
		if len(data) == 0 {
			data = json.RawMessage(`{}`)
		}
		value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err == nil {
			err = validator.Validate(value)
		}
		if err != nil {
			return toolError(fmt.Errorf("invalid arguments: %w", err)), nil
		}
		if err := ctx.Err(); err != nil {
			return toolError(err), nil
		}
		var output any
		if tool.runPath != nil {
			output, err = tool.runPath(path, data)
		} else {
			var c *collection.Collection
			c, err = collection.Load(path)
			if err == nil {
				output, err = tool.run(c.WithContext(ctx).WithEmbeddingProgress(progress), value.(map[string]any))
			}
		}
		rejected := errors.Is(err, collection.ErrSchemaIncompatible)
		if err != nil && !rejected {
			return toolError(err), nil
		}
		if rejected {
			output = struct {
				Error string `json:"error"`
				*collection.SchemaSetResult
			}{err.Error(), output.(*collection.SchemaSetResult)}
		}
		// Preserve exact JSON numbers in the structured wire response.
		encoded, err := json.Marshal(output)
		if err != nil {
			return toolError(err), nil
		}
		result := &mcp.CallToolResult{IsError: rejected, StructuredContent: json.RawMessage(encoded), Content: []mcp.Content{}}
		if rejected {
			result.Content = []mcp.Content{&mcp.TextContent{Text: string(encoded)}}
		}
		return result, nil
	})
	return nil
}

func toolError(err error) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}
	result.SetError(err)
	return result
}
