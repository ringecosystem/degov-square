package mcp

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultOAuthResourceMetadataPath = "/.well-known/oauth-protected-resource/mcp"
	defaultJWKSCacheTTL              = 5 * time.Minute
	defaultOAuthHTTPTimeout          = 10 * time.Second
)

type OAuthVerifierConfig struct {
	Issuer         string
	JWKSURL        string
	Audience       string
	Client         *http.Client
	JWKSCacheTTL   time.Duration
}

type oauthTokenVerifier struct {
	cfg OAuthVerifierConfig

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	KeyType string `json:"kty"`
	KeyID   string `json:"kid"`
	Use     string `json:"use"`
	Alg     string `json:"alg"`
	N       string `json:"n"`
	E       string `json:"e"`
}

func NewOAuthTokenVerifier(cfg OAuthVerifierConfig) sdkauth.TokenVerifier {
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: defaultOAuthHTTPTimeout}
	}
	if cfg.JWKSCacheTTL <= 0 {
		cfg.JWKSCacheTTL = defaultJWKSCacheTTL
	}

	verifier := &oauthTokenVerifier{cfg: cfg}
	return verifier.Verify
}

func OAuthAuthMiddleware(cfg Config) func(http.Handler) http.Handler {
	verifier := NewOAuthTokenVerifier(OAuthVerifierConfig{
		Issuer:   cfg.OAuthIssuer,
		JWKSURL:  cfg.OAuthJWKSURL,
		Audience: cfg.OAuthAudience,
		Client:   cfg.OAuthHTTPClient,
	})
	return sdkauth.RequireBearerToken(verifier, &sdkauth.RequireBearerTokenOptions{
		ResourceMetadataURL: oauthResourceMetadataURL(cfg),
		Scopes:              cfg.OAuthRequiredScopes,
	})
}

func (v *oauthTokenVerifier) Verify(ctx context.Context, tokenString string, _ *http.Request) (*sdkauth.TokenInfo, error) {
	if v.cfg.Issuer == "" {
		return nil, invalidOAuthToken("missing issuer configuration")
	}
	if v.cfg.JWKSURL == "" {
		return nil, invalidOAuthToken("missing JWKS URL configuration")
	}
	if v.cfg.Audience == "" {
		return nil, invalidOAuthToken("missing audience configuration")
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token missing kid")
		}
		return v.keyForKID(ctx, kid)
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithIssuer(v.cfg.Issuer), jwt.WithAudience(v.cfg.Audience), jwt.WithExpirationRequired())
	if err != nil || token == nil || !token.Valid {
		if err == nil {
			err = errors.New("token is invalid")
		}
		return nil, invalidOAuthToken(err.Error())
	}

	expiration, err := claims.GetExpirationTime()
	if err != nil || expiration == nil {
		return nil, invalidOAuthToken("token missing expiration")
	}
	subject, _ := claims.GetSubject()
	issuer, _ := claims.GetIssuer()
	audience, _ := claims.GetAudience()
	scopes := tokenScopes(claims)

	return &sdkauth.TokenInfo{
		UserID:     subject,
		Scopes:     scopes,
		Expiration: expiration.Time,
		Extra: map[string]any{
			"issuer":   issuer,
			"audience": []string(audience),
		},
	}, nil
}

func (v *oauthTokenVerifier) keyForKID(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	if fresh := v.cacheFresh(); fresh {
		key := v.keys[kid]
		v.mu.Unlock()
		if key == nil {
			return nil, fmt.Errorf("unknown key id %q", kid)
		}
		return key, nil
	}
	v.mu.Unlock()

	keys, err := v.fetchKeys(ctx)
	if err != nil {
		return nil, err
	}

	v.mu.Lock()
	v.keys = keys
	v.fetchedAt = time.Now()
	key := v.keys[kid]
	v.mu.Unlock()

	if key != nil {
		return key, nil
	}
	return nil, fmt.Errorf("unknown key id %q", kid)
}

func (v *oauthTokenVerifier) cacheFresh() bool {
	return v.keys != nil && time.Since(v.fetchedAt) < v.cfg.JWKSCacheTTL
}

func (v *oauthTokenVerifier) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := v.cfg.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("JWKS request returned status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}
	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, key := range jwks.Keys {
		publicKey, usable, err := key.rsaPublicKey()
		if err != nil {
			continue
		}
		if !usable {
			continue
		}
		keys[key.KeyID] = publicKey
	}
	if len(keys) == 0 {
		return nil, errors.New("JWKS did not contain usable RSA signing keys")
	}
	return keys, nil
}

func (k jwkKey) rsaPublicKey() (*rsa.PublicKey, bool, error) {
	if k.KeyID == "" {
		return nil, false, nil
	}
	if k.KeyType != "RSA" {
		return nil, false, nil
	}
	if k.Use != "" && k.Use != "sig" {
		return nil, false, nil
	}
	if k.Alg != "" && k.Alg != jwt.SigningMethodRS256.Alg() {
		return nil, false, nil
	}
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, false, fmt.Errorf("invalid JWK modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, false, fmt.Errorf("invalid JWK exponent: %w", err)
	}
	exponent := int(new(big.Int).SetBytes(eBytes).Int64())
	if exponent == 0 {
		return nil, false, errors.New("invalid JWK exponent")
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: exponent,
	}, true, nil
}

func NewProtectedResourceMetadataHandler(cfg Config) http.Handler {
	return sdkauth.ProtectedResourceMetadataHandler(NewProtectedResourceMetadata(cfg))
}

func RegisterProtectedResourceMetadataHandlers(mux *http.ServeMux, cfg Config) {
	handler := NewProtectedResourceMetadataHandler(cfg)
	registered := map[string]bool{}
	register := func(path string) {
		if path == "" || registered[path] {
			return
		}
		mux.Handle(path, handler)
		registered[path] = true
	}

	register("/.well-known/oauth-protected-resource")
	register(defaultOAuthResourceMetadataPath)
	register(configuredResourceMetadataPath(cfg.OAuthResourceMetadataURL))
}

func NewProtectedResourceMetadata(cfg Config) *oauthex.ProtectedResourceMetadata {
	return &oauthex.ProtectedResourceMetadata{
		Resource:               cfg.OAuthResource,
		AuthorizationServers:   cfg.OAuthAuthorizationServers,
		ScopesSupported:        cfg.OAuthScopesSupported,
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "DeGov Square MCP",
	}
}

func oauthResourceMetadataURL(cfg Config) string {
	if cfg.OAuthResourceMetadataURL != "" {
		return cfg.OAuthResourceMetadataURL
	}
	return defaultOAuthResourceMetadataPath
}

func configuredResourceMetadataPath(metadataURL string) string {
	if metadataURL == "" {
		return ""
	}
	parsed, err := url.Parse(metadataURL)
	if err != nil || parsed.Path == "" || parsed.Path[0] != '/' {
		return ""
	}
	return parsed.Path
}

func tokenScopes(claims jwt.MapClaims) []string {
	seen := map[string]bool{}
	var scopes []string
	add := func(scope string) {
		scope = strings.TrimSpace(scope)
		if scope == "" || seen[scope] {
			return
		}
		seen[scope] = true
		scopes = append(scopes, scope)
	}
	addList := func(value any) {
		switch value := value.(type) {
		case string:
			for _, scope := range strings.Fields(value) {
				add(scope)
			}
		case []string:
			for _, scope := range value {
				add(scope)
			}
		case []any:
			for _, item := range value {
				if scope, ok := item.(string); ok {
					add(scope)
				}
			}
		}
	}

	addList(claims["scope"])
	addList(claims["scp"])
	addList(claims["permissions"])
	return scopes
}

func invalidOAuthToken(message string) error {
	return fmt.Errorf("%w: %s", sdkauth.ErrInvalidToken, message)
}
