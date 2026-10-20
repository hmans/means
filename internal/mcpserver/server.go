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

const instructions = `This server works with one Markdown collection. Call collection_info
to learn its frontmatter schema before creating documents. List documents to find
IDs, then read by ID for full bodies. IDs are filename stems, without .md.
Creation validates the current schema and is not idempotent: do not retry a
successful creation. Reads and lists parse metadata without schema validation;
inspect per-document errors in list results. Schema and document edits take effect
on the next tool call. Updates, deletion, search, and schema editing are not
exposed as MCP tools yet. The optional instructions.md supplies collection-specific
workflow guidance. It is not a document and is excluded from listing and reading.
Call collection_info for current guidance; restart the server to refresh discovery
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
			name:        "collection_info",
			description: "Return the collection title, description, path, current JSON Schema for frontmatter, and usage instructions including its workflow. Call before creating documents. No arguments.",
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
			name:        "list_documents",
			description: "List IDs and frontmatter, without bodies, sorted by filename. limit defaults to 20; 0 returns all. Returns documents and has_more. Increase limit when has_more is true. Malformed documents have null frontmatter and an error; these still count toward the limit. Does not validate stored metadata against the schema.",
			schema:      `{"type":"object","properties":{"limit":{"type":"integer","minimum":0,"description":"Maximum documents; default 20, 0 for all."}},"additionalProperties":false}`,
			readOnly:    true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				limit := 20
				if value, exists := args["limit"]; exists {
					n, ok := new(big.Rat).SetString(value.(json.Number).String())
					if !ok || !n.IsInt() || !n.Num().IsInt64() {
						return nil, fmt.Errorf("limit is too large")
					}
					limit = int(n.Num().Int64())
					if int64(limit) != n.Num().Int64() {
						return nil, fmt.Errorf("limit is too large")
					}
				}
				return c.List(limit)
			},
		},
		{
			name:        "read_document",
			description: "Read a document by its filename stem (without .md). Returns id, frontmatter, and the exact Markdown body. IDs contain letters, digits, hyphens and underscores, starting with a letter or digit. Paths and symlinks are rejected. Parses metadata without validating it against the current schema.",
			schema:      `{"type":"object","properties":{"id":{"type":"string","pattern":"^[A-Za-z0-9][A-Za-z0-9_-]*$","description":"Document filename stem, without .md."}},"required":["id"],"additionalProperties":false}`,
			readOnly:    true,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				return c.Read(args["id"].(string))
			},
		},
		{
			name:        "create_document",
			description: "Create one document. Call collection_info first for its schema. Supply required frontmatter object and optional Markdown body string (default empty). Validates against the current collection schema before writing. Returns id, frontmatter, body. Generates a random 32-character hex filename stem; does not inject metadata fields or overwrite files. Each successful call creates a new document; do not retry a success.",
			schema:      `{"type":"object","properties":{"frontmatter":{"type":"object","description":"Application metadata conforming to the schema returned by collection_info."},"body":{"type":"string","description":"Markdown body, preserved exactly; defaults to empty."}},"required":["frontmatter"],"additionalProperties":false}`,
			run: func(c *collection.Collection, args map[string]any) (any, error) {
				body, _ := args["body"].(string)
				return c.Create(args["frontmatter"].(map[string]any), body)
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

type toolDefinition struct {
	name, description, schema string
	readOnly                  bool
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
			DestructiveHint: &no, OpenWorldHint: &no},
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
