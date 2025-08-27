package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/ringecosystem/degov-apps/graph"
	"github.com/ringecosystem/degov-apps/internal"
	"github.com/ringecosystem/degov-apps/internal/config"
	"github.com/ringecosystem/degov-apps/internal/directives"
	"github.com/ringecosystem/degov-apps/internal/middleware"
	"github.com/ringecosystem/degov-apps/routes"
	"github.com/ringecosystem/degov-apps/tasks"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"
)

var Version string

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

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		Debug:            config.GetAppEnv().IsDevelopment(),
	})
	httpHandler := corsHandler.Handler(mux)

	slog.Info(
		"Server is running",
		slog.String("listen", "http://::"+port+"/"),
	)
	err := http.ListenAndServe(":"+port, httpHandler)
	slog.Error("failed to listen server", "error", err)
}
