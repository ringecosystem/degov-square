package main

import (
	"context"
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
	"github.com/ringecosystem/degov-apps/services"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"
)

const defaultPort = "8080"

func main() {
	internal.AppInit()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start DAO sync scheduler in background
	daoSyncService := services.NewDaoSyncService()
	go daoSyncService.StartDaoSyncScheduler(ctx)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Received shutdown signal, gracefully shutting down...")
		cancel()
		time.Sleep(1 * time.Second) // Give time for goroutines to finish
		os.Exit(0)
	}()

	startServer()
}

func startServer() {
	cfg := config.GetConfig()
	port := cfg.GetPort()

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: graph.NewResolver()}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	mux := http.NewServeMux()

	graphiql := playground.Handler("GraphQL playground", "/graphql", playground.WithGraphiqlEnablePluginExplorer(true))
	mux.Handle("/graphiql", graphiql)
	mux.Handle("/graphql", srv)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		Debug:            config.GetAppEnv().IsDevelopment(),
	})
	handler := c.Handler(mux)

	slog.Info(
		"Server is running",
		slog.String("listen", "http://::"+port+"/"),
	)
	err := http.ListenAndServe(":"+port, handler)
	slog.Error("failed to listen server", "error", err)

}
