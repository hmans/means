// Package mcpserver exposes collection operations through MCP.
package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
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
matches literal text in titles/bodies and filters frontmatter.data by value.
IDs are stored values, independent of filenames. Lookup scans visible
subdirectories; renames preserve identity. Duplicate IDs are errors. Patches
may change title and data, never id.
Title changes preserve existing filenames.
Creation validates the current schema and is not idempotent: do not retry a
successful creation. Reads and lists validate the means envelope but not the data schema;
inspect per-document errors in list results. Schema and document edits take effect
on the next tool call. Read before update: supply its hash and only the fields to
patch. A stale hash fails; read again and review changes before retrying. Metadata
uses JSON Merge Patch (objects merge, null deletes, arrays replace); body replaces
the whole body when supplied. Deletion and schema editing are not
exposed as MCP tools yet. The optional instructions.md supplies collection-specific
workflow guidance. It is not a document and is excluded from listing and reading.
Call info for current guidance; restart the server to refresh discovery
descriptions and connection instructions after editing collection metadata.`

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
		text += "\n\nCollection workflow (instructions.md):\n" + info.Instructions
	}
	return text
}

// New validates the selected collection and builds a server bound to its path.
// Each call reloads the collection so external schema edits take effect immediately.
func New(path, version string) (*mcp.Server, error) {
	c, err := collection.Load(path)
	if err != nil {
		return nil, err
	}
	path = c.Path
	info, err := c.Describe()
	if err != nil {
		return nil, err
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "means", Title: info.Title, Description: info.Description, Version: version},
		&mcp.ServerOptions{Instructions: collectionInstructions(info)})
	for _, tool := range []toolDefinition{
		{
			name:        "info",
			description: "Return the collection title, description, path, current JSON Schema for frontmatter.data, and usage instructions including its workflow. Call before creating documents. No arguments.",
			schema:      `{"type":"object","additionalProperties":false}`,
			readOnly:    true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				current, err := c.Describe()
				if err != nil {
					return nil, err
				}
				return map[string]any{"path": c.Path, "schema": c.Schema,
					"title": current.Title, "description": current.Description,
					"instructions": collectionInstructions(current)}, nil
			},
		},
		{
			name:        "list",
			description: "List IDs and frontmatter, without bodies or hashes, sorted by filename. Use read before update. limit defaults to 20; 0 returns all. Returns documents and has_more. Increase limit when has_more is true. Scans visible subdirectories. Malformed documents and duplicate IDs have null frontmatter, an error, and a diagnostic path; id is empty if unavailable. These count toward the limit. Duplicate detection covers the entire collection. Validates the means envelope, not the application data schema.",
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
			description: "Find documents by an optional case-insensitive literal query in title or body and/or a partial filter on frontmatter.data. Filter keys combine with AND; nested objects match by supplied keys, arrays match exactly, and missing keys do not match. Results are in filename order with id, frontmatter, and a field/excerpt for text matches, but no body or hash. limit defaults to 20 matching documents; 0 returns all. has_more reports additional matches. Malformed files and duplicate IDs are omitted from matches and reported in diagnostics regardless of filters or limit. Schema-invalid application data remains searchable. Search only reads files; use read for full content before update.",
			schema:      `{"type":"object","properties":{"query":{"type":"string","description":"Optional literal substring of title or body, case-insensitive."},"filter":{"type":"object","description":"Partial match against frontmatter.data; e.g. {\"priority\":\"normal\",\"type\":\"feature\"}."},"limit":{"type":"integer","minimum":0,"description":"Maximum matches; default 20, 0 for all."}},"additionalProperties":false}`,
			readOnly:    true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				limit, err := limitArg(args)
				if err != nil {
					return nil, err
				}
				query, _ := args["query"].(string)
				filter, _ := args["filter"].(map[string]any)
				return c.Search(collection.SearchOptions{Query: query, Filter: filter, Limit: limit})
			},
		},
		{
			name:        "read",
			description: "Read a document by its stored frontmatter.id, independent of filename. Missing or duplicate IDs fail. Returns id, hash, frontmatter, and the exact Markdown body. hash is SHA-256 of the exact file bytes, required for update. IDs contain letters, digits, hyphens and underscores, starting with a letter or digit. Paths and symlinks are rejected. Validates the means envelope without enforcing the current data schema.",
			schema:      `{"type":"object","properties":{"id":{"type":"string","pattern":"^[A-Za-z0-9][A-Za-z0-9_-]*$","maxLength":128,"description":"Stored frontmatter.id, not a filename or path."}},"required":["id"],"additionalProperties":false}`,
			readOnly:    true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				return c.Read(args["id"].(string))
			},
		},
		{
			name:        "create",
			description: "Create one document. Call info first for its schema. Supply required frontmatter object and optional Markdown body string (default empty). Validates against the current collection schema before writing. Returns id, hash, frontmatter, body. Supply frontmatter.title (nonempty string) and optional frontmatter.data (object, default {}). The collection schema validates only data. Generates a unique 5-character ID in frontmatter.id and writes <id>-<title-slug>.md; do not supply id. Existing files are never overwritten. Each successful call creates a new document; do not retry a success.",
			schema:      `{"type":"object","properties":{"frontmatter":{"type":"object","properties":{"title":{"type":"string","minLength":1},"data":{"type":"object","description":"Application fields validated against the schema returned by info. Omit for {}."}},"required":["title"],"additionalProperties":false},"body":{"type":"string","description":"Markdown body, preserved exactly; defaults to empty."}},"required":["frontmatter"],"additionalProperties":false}`,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				body, _ := args["body"].(string)
				return c.Create(args["frontmatter"].(map[string]any), body)
			},
		},
		{
			name:        "update",
			description: "Patch an existing document. Read first and supply its id and hash (SHA-256 of exact stored bytes). Supply frontmatter and/or body. Frontmatter uses JSON Merge Patch: omitted keys stay, nested objects merge, null deletes keys, arrays replace. Cannot set literal null with this format. body is a replacement string; omit to retain, empty string to clear. Validates resulting metadata; stale hashes fail without writing. On conflict read again, review, and retry with the new hash. Patch only frontmatter.title and/or frontmatter.data. id is immutable and must be omitted from the patch. Title remains nonempty; data remains an object validated against the collection schema. Preserves ID, filename, and permission bits; metadata edits may reformat YAML. Returns id, new hash, frontmatter, body. Rechecks the hash before replacement; a simultaneous write between that check and replacement can still be overwritten.",
			schema:      `{"type":"object","properties":{"id":{"type":"string","pattern":"^[A-Za-z0-9][A-Za-z0-9_-]*$","maxLength":128},"hash":{"type":"string","pattern":"^[0-9a-f]{64}$","description":"Content hash returned by read."},"frontmatter":{"type":"object","properties":{"title":{"type":"string","minLength":1},"data":{"type":"object","description":"Merge patch for application data. Null values delete nested keys; arrays replace."}},"additionalProperties":false},"body":{"type":"string","description":"Replacement Markdown body; omit to retain."}},"required":["id","hash"],"anyOf":[{"required":["frontmatter"]},{"required":["body"]}],"additionalProperties":false}`,
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
	} {
		tool.description = collectionPurpose(info) + "\n\n" + tool.description
		if err := addTool(server, path, tool); err != nil {
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
	destructive               bool
	run                       func(*collection.Collection, map[string]any) (any, error)
}

func addTool(server *mcp.Server, path string, tool toolDefinition) error {
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
	no := false
	server.AddTool(&mcp.Tool{
		Name: tool.name, Description: tool.description, InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: tool.readOnly,
			DestructiveHint: &tool.destructive, OpenWorldHint: &no},
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
		c, err := collection.Load(path)
		if err != nil {
			return toolError(err), nil
		}
		output, err := tool.run(c, value.(map[string]any))
		if err != nil {
			return toolError(err), nil
		}
		encoded, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return toolError(err), nil
		}
		return &mcp.CallToolResult{
			StructuredContent: json.RawMessage(encoded),
			Content:           []mcp.Content{&mcp.TextContent{Text: string(encoded)}},
		}, nil
	})
	return nil
}

func toolError(err error) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}
	result.SetError(err)
	return result
}
