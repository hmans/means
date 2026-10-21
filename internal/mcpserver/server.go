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
subdirectories, stopping at any directory containing means.schema.json, even if invalid.
Nested collections have independent IDs, schemas, and MEANS.md guidance.
Renames within a collection preserve identity. Duplicate IDs are errors. Patches
may change title and data, never id.
Title changes preserve existing filenames.
Creation validates the current schema and is not idempotent: do not retry a
successful creation. Reads and lists validate the means envelope but not the data schema;
inspect per-document errors in list results. Schema and document edits take effect
on the next tool call. Read before update: supply its hash and only the fields to
patch. A stale hash fails; read again and review changes before retrying. Metadata
uses JSON Merge Patch (objects merge, null deletes, arrays replace); body replaces
the whole body when supplied. Deletion and schema editing are not
exposed as MCP tools yet. The optional MEANS.md supplies collection-specific
workflow guidance. It is not a document and is excluded from listing and reading.
Call info for current guidance; restart the server to refresh discovery
descriptions and connection instructions after editing collection metadata.`

const startupInstructions = `Call info for the current schema and detailed guidance.
List or search to find IDs, then read before updating with its hash. Create and
update validate frontmatter.data against the current schema. Stored IDs are not
filenames. Creation is not idempotent; do not retry a successful create.`

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
func New(path, version string, structuredOnly bool) (*mcp.Server, error) {
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
		&mcp.ServerOptions{Instructions: collectionStartupInstructions(info)})
	for _, tool := range []toolDefinition{
		{
			name:        "info",
			description: "Return current collection identity, schema, path, and full guidance. Call before creating documents.",
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
			description: "List IDs and frontmatter in filename order, without bodies or hashes. limit defaults to 20; 0 returns all. Malformed and duplicate documents appear with errors. Read an ID before updating it.",
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
			description: "Find documents by a case-insensitive literal title/body query and/or a partial frontmatter.data filter. Returns IDs, frontmatter, excerpts, and diagnostics, without bodies or hashes. limit defaults to 20; 0 returns all.",
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
			description: "Read by stored ID, independent of filename. Returns frontmatter, exact body, and the content hash required for update. Missing and duplicate IDs fail.",
			schema:      `{"type":"object","properties":{"id":{"type":"string","pattern":"^[A-Za-z0-9][A-Za-z0-9_-]*$","maxLength":128,"description":"Stored frontmatter.id, not a filename or path."}},"required":["id"],"additionalProperties":false}`,
			readOnly:    true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				return c.Read(args["id"].(string))
			},
		},
		{
			name:        "create",
			description: "Create a document with a required title, optional application data, and optional Markdown body. Call info for the data schema. Means generates the ID. Each successful call creates another document.",
			schema:      `{"type":"object","properties":{"frontmatter":{"type":"object","properties":{"title":{"type":"string","minLength":1},"data":{"type":"object","description":"Application fields validated against the schema returned by info. Omit for {}."}},"required":["title"],"additionalProperties":false},"body":{"type":"string","description":"Markdown body, preserved exactly; defaults to empty."}},"required":["frontmatter"],"additionalProperties":false}`,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				body, _ := args["body"].(string)
				return c.Create(args["frontmatter"].(map[string]any), body)
			},
		},
		{
			name:        "update",
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
	} {
		tool.description = collectionPurpose(info) + "\n\n" + tool.description
		if err := addTool(server, path, tool, structuredOnly); err != nil {
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

func addTool(server *mcp.Server, path string, tool toolDefinition, structuredOnly bool) error {
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
		// Keep JSON text for clients that do not consume structuredContent.
		// Encode it compactly once to avoid duplicating pretty-print whitespace.
		encoded, err := json.Marshal(output)
		if err != nil {
			return toolError(err), nil
		}
		result := &mcp.CallToolResult{StructuredContent: json.RawMessage(encoded), Content: []mcp.Content{}}
		if !structuredOnly {
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
