package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/ringecosystem/degov-apps/graph"
	"github.com/ringecosystem/degov-apps/internal"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"
)

const defaultPort = "8080"

func main() {
	internal.AppInit()
	startServer()
}

func startServer() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

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
		Debug:            internal.GetAppEnv().IsDevelopment(),
	})
	handler := c.Handler(mux)

	slog.Info(
		"Server is running",
		slog.String("listen", "http://::"+port+"/"),
	)
	err := http.ListenAndServe(":"+port, handler)
	slog.Error("failed to listen server", "error", err)

}
