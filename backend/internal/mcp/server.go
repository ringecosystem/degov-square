package mcp

import (
	"context"
	"log/slog"
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
	OAuthResource                    string
	OAuthResourceMetadataURL         string
	OAuthAuthorizationServers        []string
	OAuthIssuer                      string
	OAuthJWKSURL                     string
	OAuthAudience                    string
	OAuthScopesSupported             []string
	OAuthRequiredScopes              []string
	OAuthAllowStaticBearer           bool
	OAuthHTTPClient                  *http.Client
	DaoService                       daoService
	DaoConfigService                 daoConfigService
	ProposalSummaryService           proposalSummaryService
	ProposalSummaryGenerateEnabled   bool
	ProposalSummaryGenerationTimeout time.Duration
	ENSService                       ensService
	ENSResolveTimeout                time.Duration
}

type proposalSummaryService interface {
	GetCachedSummary(services.ProposalSummaryInput) (*dbmodels.ProposalSummary, error)
	GetOrGenerateSummary(services.ProposalSummaryInput) (string, error)
	GetOrGenerateSummaryWithContext(context.Context, services.ProposalSummaryInput) (string, error)
}

type ensService interface {
	Resolve(ctx context.Context, daoCode *string, address *string, name *string) (*services.ENSRecord, error)
}

func NewServer(cfg Config) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    cfg.Name,
		Version: cfg.Version,
	}, nil)

	addPingTool(server, cfg)
	addDaoTools(server, withDefaultDaoServices(cfg))
	addProposalTools(server, withDefaultENSServices(cfg))
	addIndexerTools(server, withDefaultENSServices(cfg))
	addProposalSummaryTool(server, withDefaultProposalSummaryServices(cfg))

	return server
}

func withDefaultENSServices(cfg Config) Config {
	if cfg.ENSService == nil {
		cfg.ENSService = services.NewENSService()
	}
	if cfg.ENSResolveTimeout <= 0 {
		cfg.ENSResolveTimeout = 10 * time.Second
	}
	return cfg
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

	switch cfg.AuthMode {
	case AuthModeBearer:
		if cfg.BearerToken == "" {
			slog.Error("MCP bearer token is empty; rejecting all bearer-authenticated MCP requests")
		}
		return BearerAuthMiddleware(cfg.BearerToken)(handler)
	case AuthModeOAuth:
		oauthHandler := OAuthAuthMiddleware(cfg)(handler)
		if cfg.OAuthAllowStaticBearer && cfg.BearerToken != "" {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if validBearerToken(r, cfg.BearerToken) {
					handler.ServeHTTP(w, r)
					return
				}
				oauthHandler.ServeHTTP(w, r)
			})
		}
		return oauthHandler
	case AuthModeNone:
		return handler
	default:
		slog.Error("Unsupported MCP auth mode; rejecting all MCP requests", "auth_mode", cfg.AuthMode)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		})
	}
}
