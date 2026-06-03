package mcp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestBuildHTTPHandlerOAuthRejectsMissingTokenWithResourceMetadata(t *testing.T) {
	handler := NewHTTPHandler(Config{
		Name:                     "degov-square",
		Version:                  "test",
		AuthMode:                 AuthModeOAuth,
		OAuthResourceMetadataURL: "https://mcp.example.com/.well-known/oauth-protected-resource/mcp",
		OAuthRequiredScopes:      []string{"degov.mcp.read"},
	})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	challenge := rr.Header().Get("WWW-Authenticate")
	if !strings.Contains(challenge, `resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource/mcp"`) {
		t.Fatalf("WWW-Authenticate = %q, want resource_metadata", challenge)
	}
	if !strings.Contains(challenge, `scope="degov.mcp.read"`) {
		t.Fatalf("WWW-Authenticate = %q, want scope", challenge)
	}
}

func TestBuildHTTPHandlerOAuthAllowsStaticBearerFallback(t *testing.T) {
	handler := NewHTTPHandler(Config{
		Name:                   "degov-square",
		Version:                "test",
		AuthMode:               AuthModeOAuth,
		BearerToken:            "secret",
		OAuthAllowStaticBearer: true,
	})

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code == http.StatusUnauthorized {
		t.Fatalf("status = %d, want static bearer fallback to reach MCP handler", rr.Code)
	}
}

func TestBuildHTTPHandlerOAuthRequiresExplicitStaticBearerFallback(t *testing.T) {
	handler := NewHTTPHandler(Config{
		Name:        "degov-square",
		Version:     "test",
		AuthMode:    AuthModeOAuth,
		BearerToken: "secret",
	})

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestOAuthVerifierAcceptsValidRS256JWT(t *testing.T) {
	jwks, key := newTestJWKSServer(t)
	issuer := "https://issuer.example.com"
	audience := "degov-square-mcp"
	token := signTestJWT(t, key, "test-key", jwt.MapClaims{
		"iss":   issuer,
		"sub":   "user-123",
		"aud":   audience,
		"scope": "degov.mcp.read degov.mcp.write",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"nbf":   time.Now().Add(-time.Minute).Unix(),
	})
	verifier := NewOAuthTokenVerifier(OAuthVerifierConfig{
		Issuer:   issuer,
		JWKSURL:  jwks.URL,
		Audience: audience,
		Client:   jwks.Client(),
	})

	info, err := verifier(context.Background(), token, httptest.NewRequest(http.MethodPost, "/mcp", nil))
	if err != nil {
		t.Fatalf("verifier returned error: %v", err)
	}
	if got, want := info.UserID, "user-123"; got != want {
		t.Fatalf("UserID = %q, want %q", got, want)
	}
	if got, want := info.Scopes, []string{"degov.mcp.read", "degov.mcp.write"}; !sameStrings(got, want) {
		t.Fatalf("Scopes = %v, want %v", got, want)
	}
	if info.Expiration.IsZero() {
		t.Fatal("Expiration is zero")
	}
}

func TestOAuthVerifierRejectsInvalidClaims(t *testing.T) {
	jwks, key := newTestJWKSServer(t)
	issuer := "https://issuer.example.com"
	audience := "degov-square-mcp"
	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{
			name: "wrong audience",
			claims: jwt.MapClaims{
				"iss": issuer, "sub": "user-123", "aud": "other-audience",
				"scope": "degov.mcp.read", "exp": time.Now().Add(time.Hour).Unix(),
			},
		},
		{
			name: "wrong issuer",
			claims: jwt.MapClaims{
				"iss": "https://other-issuer.example.com", "sub": "user-123", "aud": audience,
				"scope": "degov.mcp.read", "exp": time.Now().Add(time.Hour).Unix(),
			},
		},
		{
			name: "expired",
			claims: jwt.MapClaims{
				"iss": issuer, "sub": "user-123", "aud": audience,
				"scope": "degov.mcp.read", "exp": time.Now().Add(-time.Hour).Unix(),
			},
		},
	}
	verifier := NewOAuthTokenVerifier(OAuthVerifierConfig{
		Issuer:   issuer,
		JWKSURL:  jwks.URL,
		Audience: audience,
		Client:   jwks.Client(),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := signTestJWT(t, key, "test-key", tt.claims)
			if _, err := verifier(context.Background(), token, httptest.NewRequest(http.MethodPost, "/mcp", nil)); err == nil {
				t.Fatal("verifier returned nil error, want failure")
			}
		})
	}
}

func TestOAuthVerifierDoesNotRefetchFreshJWKSForUnknownKID(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	requests := 0
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeTestJWKS(t, w, []jwkKey{testJWK("test-key", key, "RSA", "sig", "RS256")})
	}))
	t.Cleanup(jwks.Close)

	issuer := "https://issuer.example.com"
	audience := "degov-square-mcp"
	verifier := NewOAuthTokenVerifier(OAuthVerifierConfig{
		Issuer:       issuer,
		JWKSURL:      jwks.URL,
		Audience:     audience,
		Client:       jwks.Client(),
		JWKSCacheTTL: time.Hour,
	})

	validToken := signTestJWT(t, key, "test-key", jwt.MapClaims{
		"iss": issuer, "sub": "user-123", "aud": audience,
		"scope": "degov.mcp.read", "exp": time.Now().Add(time.Hour).Unix(),
	})
	if _, err := verifier(context.Background(), validToken, httptest.NewRequest(http.MethodPost, "/mcp", nil)); err != nil {
		t.Fatalf("verifier returned error for valid token: %v", err)
	}

	unknownToken := signTestJWT(t, key, "unknown-key", jwt.MapClaims{
		"iss": issuer, "sub": "user-123", "aud": audience,
		"scope": "degov.mcp.read", "exp": time.Now().Add(time.Hour).Unix(),
	})
	if _, err := verifier(context.Background(), unknownToken, httptest.NewRequest(http.MethodPost, "/mcp", nil)); err == nil {
		t.Fatal("verifier returned nil error, want unknown kid failure")
	}
	if requests != 1 {
		t.Fatalf("JWKS requests = %d, want 1", requests)
	}
}

func TestOAuthVerifierSkipsIrrelevantJWKs(t *testing.T) {
	jwks, key := newTestJWKSServerWithKeys(t, func(key *rsa.PrivateKey) []jwkKey {
		return []jwkKey{
			testJWK("ec-key", key, "EC", "sig", "RS256"),
			testJWK("enc-key", key, "RSA", "enc", "RS256"),
			testJWK("wrong-alg-key", key, "RSA", "sig", "RS512"),
			testJWK("test-key", key, "RSA", "sig", "RS256"),
		}
	})
	issuer := "https://issuer.example.com"
	audience := "degov-square-mcp"
	token := signTestJWT(t, key, "test-key", jwt.MapClaims{
		"iss": issuer, "sub": "user-123", "aud": audience,
		"scope": "degov.mcp.read", "exp": time.Now().Add(time.Hour).Unix(),
	})
	verifier := NewOAuthTokenVerifier(OAuthVerifierConfig{
		Issuer:   issuer,
		JWKSURL:  jwks.URL,
		Audience: audience,
		Client:   jwks.Client(),
	})

	if _, err := verifier(context.Background(), token, httptest.NewRequest(http.MethodPost, "/mcp", nil)); err != nil {
		t.Fatalf("verifier returned error: %v", err)
	}
}

func TestOAuthVerifierSkipsMalformedJWKs(t *testing.T) {
	jwks, key := newTestJWKSServerWithKeys(t, func(key *rsa.PrivateKey) []jwkKey {
		malformed := testJWK("bad-key", key, "RSA", "sig", "RS256")
		malformed.N = "not-base64"
		return []jwkKey{
			malformed,
			testJWK("test-key", key, "RSA", "sig", "RS256"),
		}
	})
	issuer := "https://issuer.example.com"
	audience := "degov-square-mcp"
	token := signTestJWT(t, key, "test-key", jwt.MapClaims{
		"iss": issuer, "sub": "user-123", "aud": audience,
		"scope": "degov.mcp.read", "exp": time.Now().Add(time.Hour).Unix(),
	})
	verifier := NewOAuthTokenVerifier(OAuthVerifierConfig{
		Issuer:   issuer,
		JWKSURL:  jwks.URL,
		Audience: audience,
		Client:   jwks.Client(),
	})

	if _, err := verifier(context.Background(), token, httptest.NewRequest(http.MethodPost, "/mcp", nil)); err != nil {
		t.Fatalf("verifier returned error: %v", err)
	}
}

func TestBuildHTTPHandlerOAuthRejectsMissingScope(t *testing.T) {
	jwks, key := newTestJWKSServer(t)
	issuer := "https://issuer.example.com"
	audience := "degov-square-mcp"
	token := signTestJWT(t, key, "test-key", jwt.MapClaims{
		"iss":   issuer,
		"sub":   "user-123",
		"aud":   audience,
		"scope": "degov.mcp.write",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	handler := NewHTTPHandler(Config{
		Name:                   "degov-square",
		Version:                "test",
		AuthMode:               AuthModeOAuth,
		OAuthIssuer:            issuer,
		OAuthJWKSURL:           jwks.URL,
		OAuthAudience:          audience,
		OAuthRequiredScopes:    []string{"degov.mcp.read"},
		OAuthAllowStaticBearer: false,
		OAuthHTTPClient:        jwks.Client(),
	})

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestProtectedResourceMetadataHandlerReturnsOAuthMetadata(t *testing.T) {
	handler := NewProtectedResourceMetadataHandler(Config{
		OAuthResource:             "https://mcp.example.com/mcp",
		OAuthAuthorizationServers: []string{"https://issuer.example.com"},
		OAuthScopesSupported:      []string{"degov.mcp.read"},
	})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/mcp", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var got struct {
		Resource               string   `json:"resource"`
		AuthorizationServers   []string `json:"authorization_servers"`
		ScopesSupported        []string `json:"scopes_supported"`
		BearerMethodsSupported []string `json:"bearer_methods_supported"`
		ResourceName           string   `json:"resource_name"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("metadata JSON decode failed: %v", err)
	}
	if got.Resource != "https://mcp.example.com/mcp" {
		t.Fatalf("resource = %q, want configured resource", got.Resource)
	}
	if !sameStrings(got.AuthorizationServers, []string{"https://issuer.example.com"}) {
		t.Fatalf("authorization_servers = %v, want issuer", got.AuthorizationServers)
	}
	if !sameStrings(got.ScopesSupported, []string{"degov.mcp.read"}) {
		t.Fatalf("scopes_supported = %v, want read scope", got.ScopesSupported)
	}
	if !sameStrings(got.BearerMethodsSupported, []string{"header"}) {
		t.Fatalf("bearer_methods_supported = %v, want header", got.BearerMethodsSupported)
	}
	if got.ResourceName != "DeGov Square MCP" {
		t.Fatalf("resource_name = %q, want DeGov Square MCP", got.ResourceName)
	}
}

func TestRegisterProtectedResourceMetadataHandlersRegistersWellKnownPaths(t *testing.T) {
	mux := http.NewServeMux()
	RegisterProtectedResourceMetadataHandlers(mux, Config{
		OAuthResource:             "https://mcp.example.com/mcp",
		OAuthAuthorizationServers: []string{"https://issuer.example.com"},
		OAuthScopesSupported:      []string{"degov.mcp.read"},
	})

	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp"} {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, rr.Code, http.StatusOK)
		}
	}
}

func TestRegisterProtectedResourceMetadataHandlersRegistersConfiguredMetadataPath(t *testing.T) {
	mux := http.NewServeMux()
	RegisterProtectedResourceMetadataHandlers(mux, Config{
		OAuthResourceMetadataURL:  "https://mcp.example.com/oauth/resource-metadata",
		OAuthResource:             "https://mcp.example.com/mcp",
		OAuthAuthorizationServers: []string{"https://issuer.example.com"},
		OAuthScopesSupported:      []string{"degov.mcp.read"},
	})

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/oauth/resource-metadata", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestRegisterProtectedResourceMetadataHandlersIgnoresInvalidMetadataURL(t *testing.T) {
	mux := http.NewServeMux()
	RegisterProtectedResourceMetadataHandlers(mux, Config{
		OAuthResourceMetadataURL:  "://bad-url",
		OAuthResource:             "https://mcp.example.com/mcp",
		OAuthAuthorizationServers: []string{"https://issuer.example.com"},
		OAuthScopesSupported:      []string{"degov.mcp.read"},
	})

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/mcp", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("well-known status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func newTestJWKSServer(t *testing.T) (*httptest.Server, *rsa.PrivateKey) {
	return newTestJWKSServerWithKeys(t, func(key *rsa.PrivateKey) []jwkKey {
		return []jwkKey{testJWK("test-key", key, "RSA", "sig", "RS256")}
	})
}

func newTestJWKSServerWithKeys(t *testing.T, keys func(*rsa.PrivateKey) []jwkKey) (*httptest.Server, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJWKS(t, w, keys(key))
	}))
	t.Cleanup(server.Close)
	return server, key
}

func writeTestJWKS(t *testing.T, w http.ResponseWriter, keys []jwkKey) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"keys": keys}); err != nil {
		t.Fatalf("jwks encode error = %v", err)
	}
}

func testJWK(kid string, key *rsa.PrivateKey, keyType string, use string, alg string) jwkKey {
	return jwkKey{
		KeyType: keyType,
		Use:     use,
		Alg:     alg,
		KeyID:   kid,
		N:       base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
		E:       base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
	}
}

func signTestJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return signed
}

func sameStrings(got []string, want []string) bool {
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
