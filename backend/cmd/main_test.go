package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/internal/middleware"
)

func TestCORSAllowsMCPStreamableHTTPHeaders(t *testing.T) {
	handler := newCORSHandler().Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	req.Header.Set("Origin", "https://inspector.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "accept,authorization,content-type,last-event-id,mcp-protocol-version,mcp-session-id")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected preflight status %d, got %d", http.StatusNoContent, rec.Code)
	}

	allowedHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	for _, header := range []string{"accept", "authorization", "content-type", "last-event-id", "mcp-protocol-version", "mcp-session-id"} {
		if !strings.Contains(allowedHeaders, header) {
			t.Fatalf("expected Access-Control-Allow-Headers to contain %q, got %q", header, allowedHeaders)
		}
	}
}

func TestOpenAIAppsChallengeRoute(t *testing.T) {
	mux := http.NewServeMux()
	registerOpenAIAppsChallengeRoute(mux)

	req := httptest.NewRequest(http.MethodGet, openAIAppsChallengePath, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing token status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	t.Setenv(openAIAppsChallengeTokenEnv, "challenge-token")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("challenge status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "challenge-token" {
		t.Fatalf("challenge body = %q, want %q", got, "challenge-token")
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("content type = %q, want text/plain", got)
	}
}

func TestRegisterStytchOAuthRoutesOnlyWhenEnabled(t *testing.T) {
	t.Setenv("MCP_STYTCH_OAUTH_ENABLED", "false")
	if err := config.InitConfig(); err != nil {
		t.Fatalf("InitConfig: %v", err)
	}

	mux := http.NewServeMux()
	registerStytchOAuthRoutes(mux, middleware.NewChain(), config.GetConfig(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/stytch/authorize/start", strings.NewReader(`{}`))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("disabled route status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	t.Setenv("MCP_STYTCH_OAUTH_ENABLED", "true")
	t.Setenv("MCP_STYTCH_OAUTH_DOMAIN", "https://test.stytch.com")
	t.Setenv("MCP_STYTCH_OAUTH_PROJECT_ID", "project-test")
	t.Setenv("MCP_STYTCH_OAUTH_SECRET", "secret-test")
	if err := config.InitConfig(); err != nil {
		t.Fatalf("InitConfig: %v", err)
	}

	mux = http.NewServeMux()
	registerStytchOAuthRoutes(mux, middleware.NewChain(), config.GetConfig(), nil)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/oauth/stytch/authorize/start", strings.NewReader(`{}`))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("enabled route status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
