package config

import "testing"

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
}

func TestMCPConfigReadsEnvironment(t *testing.T) {
	t.Setenv("MCP_ENABLED", "true")
	t.Setenv("MCP_PATH", "/api/mcp")
	t.Setenv("MCP_AUTH_MODE", "none")
	t.Setenv("MCP_BEARER_TOKEN", "test-token")
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
}
