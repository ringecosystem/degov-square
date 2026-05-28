package mcp

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBearerAuthMiddlewareRejectsMissingToken(t *testing.T) {
	handler := BearerAuthMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if got, want := rr.Header().Get("WWW-Authenticate"), "Bearer"; got != want {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, want)
	}
}

func TestBearerAuthMiddlewareAllowsMatchingToken(t *testing.T) {
	called := false
	handler := BearerAuthMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestBuildHTTPHandlerUsesBearerAuth(t *testing.T) {
	handler := NewHTTPHandler(Config{
		Name:        "degov-square",
		Version:     "test",
		AuthMode:    AuthModeBearer,
		BearerToken: "secret",
	})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestBuildHTTPHandlerAllowsExplicitNoAuth(t *testing.T) {
	handler := NewHTTPHandler(Config{
		Name:     "degov-square",
		Version:  "test",
		AuthMode: AuthModeNone,
	})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rr.Code == http.StatusUnauthorized {
		t.Fatalf("status = %d, want non-auth response", rr.Code)
	}
}

func TestBuildHTTPHandlerRejectsEmptyBearerToken(t *testing.T) {
	var logs bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	handler := NewHTTPHandler(Config{
		Name:     "degov-square",
		Version:  "test",
		AuthMode: AuthModeBearer,
	})

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(logs.String(), "MCP bearer token is empty") {
		t.Fatalf("logs = %q, want empty token diagnostic", logs.String())
	}
}

func TestBuildHTTPHandlerRejectsUnknownAuthMode(t *testing.T) {
	var logs bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	handler := NewHTTPHandler(Config{
		Name:     "degov-square",
		Version:  "test",
		AuthMode: "bearer-token",
	})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(logs.String(), "Unsupported MCP auth mode") {
		t.Fatalf("logs = %q, want unsupported auth mode diagnostic", logs.String())
	}
}
