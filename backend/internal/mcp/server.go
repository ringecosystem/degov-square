package mcp

import (
	"net/http"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"github.com/ringecosystem/degov-square/services"
)

type Config struct {
	Name                             string
	Version                          string
	AuthMode                         string
	BearerToken                      string
	DaoService                       daoService
	DaoConfigService                 daoConfigService
	ProposalSummaryService           proposalSummaryService
	ProposalSummaryGenerateEnabled   bool
	ProposalSummaryGenerationTimeout time.Duration
}

type proposalSummaryService interface {
	GetCachedSummary(services.ProposalSummaryInput) (*dbmodels.ProposalSummary, error)
	GetOrGenerateSummary(services.ProposalSummaryInput) (string, error)
}

func NewServer(cfg Config) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    cfg.Name,
		Version: cfg.Version,
	}, nil)

	addPingTool(server, cfg)
	addDaoTools(server, withDefaultDaoServices(cfg))
	addProposalTools(server)
	addProposalSummaryTool(server, withDefaultProposalSummaryServices(cfg))

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

func withDefaultProposalSummaryServices(cfg Config) Config {
	if cfg.ProposalSummaryService == nil {
		cfg.ProposalSummaryService = services.NewProposalSummaryService()
	}
	if cfg.ProposalSummaryGenerationTimeout <= 0 {
		cfg.ProposalSummaryGenerationTimeout = 30 * time.Second
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
