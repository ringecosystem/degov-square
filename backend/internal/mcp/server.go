package mcp

import (
	"net/http"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config struct {
	Name        string
	Version     string
	AuthMode    string
	BearerToken string
}

func NewServer(cfg Config) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    cfg.Name,
		Version: cfg.Version,
	}, nil)

	addPingTool(server, cfg)
	addProposalTools(server)

	return server
}

func NewHTTPHandler(cfg Config) http.Handler {
	handler := sdkmcp.NewStreamableHTTPHandler(func(r *http.Request) *sdkmcp.Server {
		return NewServer(cfg)
	}, nil)

	if cfg.AuthMode == AuthModeBearer {
		return BearerAuthMiddleware(cfg.BearerToken)(handler)
	}

	return handler
}
