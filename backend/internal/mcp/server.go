package mcp

import (
	"net/http"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ringecosystem/degov-square/services"
)

type Config struct {
	Name             string
	Version          string
	AuthMode         string
	BearerToken      string
	DaoService       daoService
	DaoConfigService daoConfigService
}

func NewServer(cfg Config) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    cfg.Name,
		Version: cfg.Version,
	}, nil)

	addPingTool(server, cfg)
	addDaoTools(server, withDefaultDaoServices(cfg))

	return server
}

func withDefaultDaoServices(cfg Config) Config {
	if cfg.DaoService == nil {
		cfg.DaoService = services.NewDaoService()
	}
	if cfg.DaoConfigService == nil {
		cfg.DaoConfigService = services.NewDaoConfigService()
	}
	return cfg
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
