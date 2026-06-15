package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/ringecosystem/degov-square/graph"
	"github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/internal/directives"
	mcpserver "github.com/ringecosystem/degov-square/internal/mcp"
	"github.com/ringecosystem/degov-square/internal/middleware"
	"github.com/ringecosystem/degov-square/routes"
	"github.com/ringecosystem/degov-square/tasks"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"
)

var Version string

const (
	openAIAppsChallengePath     = "/.well-known/openai-apps-challenge"
	openAIAppsChallengeTokenEnv = "OPENAI_APPS_CHALLENGE_TOKEN"
)

func main() {
	if Version == "" {
		fmt.Println("Version: Debug")
	} else {
		fmt.Println("Version:", Version)
	}

	// Initialize the application
	internal.AppInit()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background tasks
	startBackgroundTasks(ctx)

	// Handle graceful shutdown
	go handleGracefulShutdown(cancel)

	// Start the web server
	startServer()
}

// startBackgroundTasks starts all background tasks
func startBackgroundTasks(ctx context.Context) {
	slog.Info("Starting background tasks...")

	// Create task manager
	taskManager, err := tasks.NewTaskManager()
	if err != nil {
		slog.Error("Failed to create task manager", "error", err)
		return
	}

	// Get task definitions (combines config and constructor)
	taskDefinitions := tasks.GetTaskDefinitions()

	// Register tasks based on definitions
	registeredCount := 0
	for _, def := range taskDefinitions {
		if !def.Config.Enabled {
			slog.Info("Task disabled, skipping", "task", def.Config.Name)
			continue
		}

		task := def.Constructor()
		if err := taskManager.RegisterTask(task, def.Config.Interval); err != nil {
			slog.Error("Failed to register task", "task", def.Config.Name, "error", err)
			continue
		}

		registeredCount++
	}

	slog.Info("Registered tasks",
		"total_count", registeredCount,
		"registered_tasks", taskManager.ListTasks())

	// Start the task manager
	go taskManager.Start(ctx)

	slog.Info("Background tasks started successfully")
}

// handleGracefulShutdown handles graceful shutdown signals
func handleGracefulShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	slog.Info("Received shutdown signal, gracefully shutting down...")
	cancel()
	time.Sleep(2 * time.Second) // Give time for background tasks to finish
	os.Exit(0)
}

// startServer starts the GraphQL server
func startServer() {
	cfg := config.GetConfig()
	port := cfg.GetPort()

	// Configure directives
	graphqlConfig := graph.Config{
		Resolvers: graph.NewResolver(),
		Directives: graph.DirectiveRoot{
			Auth:      directives.AuthDirective,
			Authorize: directives.AuthorizeDirective,
		},
	}

	gqlSrv := handler.New(graph.NewExecutableSchema(graphqlConfig))

	gqlSrv.AddTransport(transport.Options{})
	gqlSrv.AddTransport(transport.GET{})
	gqlSrv.AddTransport(transport.POST{})

	gqlSrv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	gqlSrv.Use(extension.Introspection{})
	gqlSrv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	// Create middleware chain including auth middleware
	middlewareChain := middleware.NewChain(
		middleware.RecoveryMiddleware(), // Recovery should be first
		middleware.LoggingMiddleware(),  // Logging
		// middleware.SecurityHeadersMiddleware(),      // Security headers
		middleware.NewDegovMiddleware().Middleware(),
		middleware.NewAuthMiddleware().Middleware(), // Authentication
	)

	mux := http.NewServeMux()
	registerOpenAIAppsChallengeRoute(mux)

	graphiql := playground.Handler("GraphQL playground", "/graphql", playground.WithGraphiqlEnablePluginExplorer(true))
	mux.Handle("/graphiql", graphiql)

	// Apply complete middleware chain to GraphQL endpoint
	graphqlHandler := middlewareChain.Then(gqlSrv)
	mux.Handle("/graphql", graphqlHandler)

	// Create DAO route handler
	daoRoute := routes.NewDaoRoute()

	// Support both patterns: /dao/config and /dao/config/{dao}
	mux.Handle("/dao/config", middlewareChain.Then(http.HandlerFunc(daoRoute.ConfigHandler)))
	mux.Handle("/dao/config/{dao}", middlewareChain.Then(http.HandlerFunc(daoRoute.ConfigHandler)))

	registerStytchOAuthRoutes(mux, middlewareChain, cfg, nil)

	if cfg.GetMCPEnabled() {
		if mcpserver.AuthModeIncludes(cfg.GetMCPAuthMode(), mcpserver.AuthModeOAuth) {
			mcpserver.RegisterProtectedResourceMetadataHandlers(mux, mcpserver.Config{
				OAuthResource:             cfg.GetMCPOAuthResource(),
				OAuthResourceMetadataURL:  cfg.GetMCPOAuthResourceMetadataURL(),
				OAuthAuthorizationServers: cfg.GetMCPOAuthAuthorizationServers(),
				OAuthScopesSupported:      cfg.GetMCPOAuthScopesSupported(),
			})
		}
		mux.Handle(cfg.GetMCPPath(), mcpserver.NewHTTPHandler(mcpserver.Config{
			Name:                             "degov-square",
			Version:                          getMCPVersion(),
			AuthMode:                         cfg.GetMCPAuthMode(),
			BearerToken:                      cfg.GetMCPBearerToken(),
			OAuthResource:                    cfg.GetMCPOAuthResource(),
			OAuthResourceMetadataURL:         cfg.GetMCPOAuthResourceMetadataURL(),
			OAuthAuthorizationServers:        cfg.GetMCPOAuthAuthorizationServers(),
			OAuthIssuer:                      cfg.GetMCPOAuthIssuer(),
			OAuthJWKSURL:                     cfg.GetMCPOAuthJWKSURL(),
			OAuthAudience:                    cfg.GetMCPOAuthAudience(),
			OAuthScopesSupported:             cfg.GetMCPOAuthScopesSupported(),
			OAuthRequiredScopes:              cfg.GetMCPOAuthRequiredScopes(),
			OAuthAllowStaticBearer:           cfg.GetMCPOAuthAllowStaticBearer(),
			ProposalSummaryGenerateEnabled:   cfg.GetMCPProposalSummaryGenerateEnabled(),
			ProposalSummaryGenerationTimeout: cfg.GetMCPProposalSummaryTimeout(),
		}))
	}

	httpHandler := newCORSHandler().Handler(mux)

	slog.Info(
		"Server is running",
		slog.String("listen", "http://::"+port+"/"),
	)
	err := http.ListenAndServe(":"+port, httpHandler)
	slog.Error("failed to listen server", "error", err)
}

func registerOpenAIAppsChallengeRoute(mux *http.ServeMux) {
	mux.HandleFunc(openAIAppsChallengePath, openAIAppsChallengeHandler)
}

func openAIAppsChallengeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimSpace(os.Getenv(openAIAppsChallengeTokenEnv))
	if token == "" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(token))
}

func registerStytchOAuthRoutes(mux *http.ServeMux, middlewareChain *middleware.Chain, cfg *config.Config, httpClient *http.Client) {
	if !cfg.GetMCPStytchOAuthEnabled() {
		return
	}

	client := mcpserver.NewStytchOAuthClient(mcpserver.StytchOAuthClientConfig{
		Domain:     cfg.GetMCPStytchOAuthDomain(),
		ProjectID:  cfg.GetMCPStytchOAuthProjectID(),
		Secret:     cfg.GetMCPStytchOAuthSecret(),
		HTTPClient: httpClient,
	})
	handler := mcpserver.NewStytchOAuthHandler(mcpserver.StytchOAuthHandlerConfig{
		Client:        client,
		UserIDPrefix:  cfg.GetMCPStytchOAuthUserIDPrefix(),
		OAuthResource: cfg.GetMCPOAuthResource(),
	})
	mux.Handle("/api/oauth/stytch/authorize/start", middlewareChain.Then(http.HandlerFunc(handler.AuthorizeStart)))
	mux.Handle("/api/oauth/stytch/authorize/submit", middlewareChain.Then(http.HandlerFunc(handler.AuthorizeSubmit)))
}

func getMCPVersion() string {
	if Version == "" {
		return "Debug"
	}

	return Version
}

func newCORSHandler() *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"Last-Event-ID",
			"MCP-Protocol-Version",
			"Mcp-Session-Id",
		},
		AllowCredentials: true,
		Debug:            config.GetAppEnv().IsDevelopment(),
	})
}
