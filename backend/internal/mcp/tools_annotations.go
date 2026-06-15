package mcp

import sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

func readOnlyToolAnnotations() *sdkmcp.ToolAnnotations {
	return &sdkmcp.ToolAnnotations{
		ReadOnlyHint:    true,
		OpenWorldHint:   boolPtr(false),
		DestructiveHint: boolPtr(false),
	}
}

func boolPtr(value bool) *bool {
	return &value
}
