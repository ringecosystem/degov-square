package config

import (
	"testing"
	"time"
)

func TestMCPConfigDefaults(t *testing.T) {
	globalConfig = nil
	t.Cleanup(func() { globalConfig = nil })

	cfg := GetConfig()

	if cfg.GetMCPEnabled() {
		t.Fatal("GetMCPEnabled() = true, want false")
	}
	if got, want := cfg.GetMCPPath(), "/mcp"; got != want {
		t.Fatalf("GetMCPPath() = %q, want %q", got, want)
	}
	if got, want := cfg.GetMCPAuthMode(), "bearer"; got != want {
		t.Fatalf("GetMCPAuthMode() = %q, want %q", got, want)
	}
	if got := cfg.GetMCPBearerToken(); got != "" {
		t.Fatalf("GetMCPBearerToken() = %q, want empty", got)
	}
	if got := cfg.GetMCPOAuthResource(); got != "" {
		t.Fatalf("GetMCPOAuthResource() = %q, want empty", got)
	}
	if got := cfg.GetMCPOAuthResourceMetadataURL(); got != "" {
		t.Fatalf("GetMCPOAuthResourceMetadataURL() = %q, want empty", got)
	}
	if got := cfg.GetMCPOAuthAuthorizationServers(); len(got) != 0 {
		t.Fatalf("GetMCPOAuthAuthorizationServers() = %v, want empty", got)
	}
	if got := cfg.GetMCPOAuthIssuer(); got != "" {
		t.Fatalf("GetMCPOAuthIssuer() = %q, want empty", got)
	}
	if got := cfg.GetMCPOAuthJWKSURL(); got != "" {
		t.Fatalf("GetMCPOAuthJWKSURL() = %q, want empty", got)
	}
	if got := cfg.GetMCPOAuthAudience(); got != "" {
		t.Fatalf("GetMCPOAuthAudience() = %q, want empty", got)
	}
	if got, want := cfg.GetMCPOAuthScopesSupported(), []string{"degov.mcp.read"}; !equalStrings(got, want) {
		t.Fatalf("GetMCPOAuthScopesSupported() = %v, want %v", got, want)
	}
	if got, want := cfg.GetMCPOAuthRequiredScopes(), []string{"degov.mcp.read"}; !equalStrings(got, want) {
		t.Fatalf("GetMCPOAuthRequiredScopes() = %v, want %v", got, want)
	}
	if cfg.GetMCPOAuthAllowStaticBearer() {
		t.Fatal("GetMCPOAuthAllowStaticBearer() = true, want false")
	}
	if cfg.GetMCPProposalSummaryGenerateEnabled() {
		t.Fatal("GetMCPProposalSummaryGenerateEnabled() = true, want false")
	}
	if got, want := cfg.GetMCPProposalSummaryTimeout(), 30*time.Second; got != want {
		t.Fatalf("GetMCPProposalSummaryTimeout() = %s, want 30s", got)
	}
}

func TestMCPConfigReadsEnvironment(t *testing.T) {
	t.Setenv("MCP_ENABLED", "true")
	t.Setenv("MCP_PATH", "/api/mcp")
	t.Setenv("MCP_AUTH_MODE", "none")
	t.Setenv("MCP_BEARER_TOKEN", "test-token")
	t.Setenv("MCP_OAUTH_RESOURCE", "https://mcp.example.com/mcp")
	t.Setenv("MCP_OAUTH_RESOURCE_METADATA_URL", "https://mcp.example.com/.well-known/oauth-protected-resource/mcp")
	t.Setenv("MCP_OAUTH_AUTHORIZATION_SERVERS", "https://issuer.example.com, https://issuer-2.example.com")
	t.Setenv("MCP_OAUTH_ISSUER", "https://issuer.example.com")
	t.Setenv("MCP_OAUTH_JWKS_URL", "https://issuer.example.com/.well-known/jwks.json")
	t.Setenv("MCP_OAUTH_AUDIENCE", "degov-square-mcp")
	t.Setenv("MCP_OAUTH_SCOPES_SUPPORTED", "degov.mcp.read, degov.mcp.write")
	t.Setenv("MCP_OAUTH_REQUIRED_SCOPES", "degov.mcp.read")
	t.Setenv("MCP_OAUTH_ALLOW_STATIC_BEARER", "false")
	t.Setenv("MCP_PROPOSAL_SUMMARY_GENERATE_ENABLED", "true")
	t.Setenv("MCP_PROPOSAL_SUMMARY_TIMEOUT", "5s")
	globalConfig = nil
	t.Cleanup(func() { globalConfig = nil })

	cfg := GetConfig()

	if !cfg.GetMCPEnabled() {
		t.Fatal("GetMCPEnabled() = false, want true")
	}
	if got, want := cfg.GetMCPPath(), "/api/mcp"; got != want {
		t.Fatalf("GetMCPPath() = %q, want %q", got, want)
	}
	if got, want := cfg.GetMCPAuthMode(), "none"; got != want {
		t.Fatalf("GetMCPAuthMode() = %q, want %q", got, want)
	}
	if got, want := cfg.GetMCPBearerToken(), "test-token"; got != want {
		t.Fatalf("GetMCPBearerToken() = %q, want %q", got, want)
	}
	if got, want := cfg.GetMCPOAuthResource(), "https://mcp.example.com/mcp"; got != want {
		t.Fatalf("GetMCPOAuthResource() = %q, want %q", got, want)
	}
	if got, want := cfg.GetMCPOAuthResourceMetadataURL(), "https://mcp.example.com/.well-known/oauth-protected-resource/mcp"; got != want {
		t.Fatalf("GetMCPOAuthResourceMetadataURL() = %q, want %q", got, want)
	}
	if got, want := cfg.GetMCPOAuthAuthorizationServers(), []string{"https://issuer.example.com", "https://issuer-2.example.com"}; !equalStrings(got, want) {
		t.Fatalf("GetMCPOAuthAuthorizationServers() = %v, want %v", got, want)
	}
	if got, want := cfg.GetMCPOAuthIssuer(), "https://issuer.example.com"; got != want {
		t.Fatalf("GetMCPOAuthIssuer() = %q, want %q", got, want)
	}
	if got, want := cfg.GetMCPOAuthJWKSURL(), "https://issuer.example.com/.well-known/jwks.json"; got != want {
		t.Fatalf("GetMCPOAuthJWKSURL() = %q, want %q", got, want)
	}
	if got, want := cfg.GetMCPOAuthAudience(), "degov-square-mcp"; got != want {
		t.Fatalf("GetMCPOAuthAudience() = %q, want %q", got, want)
	}
	if got, want := cfg.GetMCPOAuthScopesSupported(), []string{"degov.mcp.read", "degov.mcp.write"}; !equalStrings(got, want) {
		t.Fatalf("GetMCPOAuthScopesSupported() = %v, want %v", got, want)
	}
	if got, want := cfg.GetMCPOAuthRequiredScopes(), []string{"degov.mcp.read"}; !equalStrings(got, want) {
		t.Fatalf("GetMCPOAuthRequiredScopes() = %v, want %v", got, want)
	}
	if cfg.GetMCPOAuthAllowStaticBearer() {
		t.Fatal("GetMCPOAuthAllowStaticBearer() = true, want false")
	}
	if !cfg.GetMCPProposalSummaryGenerateEnabled() {
		t.Fatal("GetMCPProposalSummaryGenerateEnabled() = false, want true")
	}
	if got, want := cfg.GetMCPProposalSummaryTimeout(), 5*time.Second; got != want {
		t.Fatalf("GetMCPProposalSummaryTimeout() = %s, want 5s", got)
	}
}

func equalStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
