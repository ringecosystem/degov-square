package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
