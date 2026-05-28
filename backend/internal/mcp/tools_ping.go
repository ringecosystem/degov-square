package mcp

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type pingInput struct{}

type pingOutput struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

func addPingTool(server *sdkmcp.Server, cfg Config) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "ping",
		Title:       "Ping",
		Description: "Return the MCP service health status.",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input pingInput) (*sdkmcp.CallToolResult, pingOutput, error) {
		return nil, pingOutput{
			Status:  "ok",
			Service: cfg.Name,
			Version: cfg.Version,
		}, nil
	})
}
