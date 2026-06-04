package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ringecosystem/degov-square/internal/middleware"
	"github.com/ringecosystem/degov-square/types"
)

func TestStytchOAuthClientConsumerStartRequest(t *testing.T) {
	allowUnsafeStytchOAuthTestDomain(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/v1/idp/oauth/authorize/start"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		if got, want := r.Method, http.MethodPost; got != want {
			t.Fatalf("method = %q, want %q", got, want)
		}
		projectID, secret, ok := r.BasicAuth()
		if !ok || projectID != "project-test" || secret != "secret-test" {
			t.Fatalf("basic auth = %q/%q/%v, want project-test/secret-test/true", projectID, secret, ok)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		assertJSONValue(t, body, "client_id", "client-test")
		assertJSONValue(t, body, "redirect_uri", "https://client.example/callback")
		assertJSONValue(t, body, "response_type", "code")
		assertJSONValue(t, body, "user_id", "degov-square:user-123")
		assertJSONArray(t, body, "scopes", []string{"openid", "degov.mcp.read"})
		if _, ok := body["resources"]; ok {
			t.Fatal("start request included resources; want resources only on submit")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"client":{"client_id":"client-test","client_name":"Test App"},"consent_required":true,"scope_results":[{"scope":"openid","description":"Profile","is_grantable":true}],"status_code":200}`))
	}))
	defer server.Close()

	client := NewStytchOAuthClient(StytchOAuthClientConfig{
		Domain:    server.URL,
		ProjectID: "project-test",
		Secret:    "secret-test",
	})

	resp, err := client.AuthorizeStart(context.Background(), StytchOAuthAuthorizeStartRequest{
		StytchOAuthAuthorizeRequest: StytchOAuthAuthorizeRequest{
			ClientID:     "client-test",
			RedirectURI:  "https://client.example/callback",
			ResponseType: "code",
			Scopes:       []string{"openid", "degov.mcp.read"},
			UserID:       "degov-square:user-123",
		},
	})
	if err != nil {
		t.Fatalf("AuthorizeStart returned error: %v", err)
	}
	if resp.Client.ClientName != "Test App" {
		t.Fatalf("client_name = %q, want Test App", resp.Client.ClientName)
	}
	if !resp.ConsentRequired {
		t.Fatal("consent_required = false, want true")
	}
}

func TestStytchOAuthClientConsumerSubmitRequest(t *testing.T) {
	allowUnsafeStytchOAuthTestDomain(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/v1/idp/oauth/authorize"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		assertJSONValue(t, body, "client_id", "client-test")
		assertJSONValue(t, body, "state", "state-test")
		assertJSONValue(t, body, "nonce", "nonce-test")
		assertJSONValue(t, body, "code_challenge", "challenge-test")
		assertJSONValue(t, body, "code_challenge_method", "S256")
		assertJSONValue(t, body, "consent_granted", true)
		assertJSONArray(t, body, "resources", []string{"https://square.degov.ai/mcp"})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"redirect_uri":"https://client.example/callback?code=abc","status_code":200}`))
	}))
	defer server.Close()

	client := NewStytchOAuthClient(StytchOAuthClientConfig{
		Domain:    server.URL,
		ProjectID: "project-test",
		Secret:    "secret-test",
	})

	resp, err := client.AuthorizeSubmit(context.Background(), StytchOAuthAuthorizeSubmitRequest{
		StytchOAuthAuthorizeRequest: StytchOAuthAuthorizeRequest{
			ClientID:            "client-test",
			RedirectURI:         "https://client.example/callback",
			ResponseType:        "code",
			Scopes:              []string{"openid"},
			UserID:              "user-test",
			State:               "state-test",
			Nonce:               "nonce-test",
			CodeChallenge:       "challenge-test",
			CodeChallengeMethod: "S256",
			Resources:           []string{"https://square.degov.ai/mcp"},
		},
		ConsentGranted: true,
	})
	if err != nil {
		t.Fatalf("AuthorizeSubmit returned error: %v", err)
	}
	if got, want := resp.RedirectURI, "https://client.example/callback?code=abc"; got != want {
		t.Fatalf("redirect_uri = %q, want %q", got, want)
	}
}

func TestStytchOAuthHandlerStartUnauthorized(t *testing.T) {
	handler := NewStytchOAuthHandler(StytchOAuthHandlerConfig{
		Client: &fakeStytchOAuthClient{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/oauth/stytch/authorize/start", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	handler.AuthorizeStart(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestStytchOAuthHandlerSubmitReturnsRedirectURI(t *testing.T) {
	client := &fakeStytchOAuthClient{
		submitResponse: StytchOAuthAuthorizeSubmitResponse{
			RedirectURI: "https://client.example/callback?code=abc",
		},
	}
	handler := NewStytchOAuthHandler(StytchOAuthHandlerConfig{
		Client:        client,
		UserIDPrefix:  "degov-square:",
		OAuthResource: "https://square.degov.ai/mcp",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/oauth/stytch/authorize/submit", bytes.NewReader([]byte(`{"client_id":"client-test","redirect_uri":"https://client.example/callback","scope":"openid degov.mcp.read","consent_granted":true}`)))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserClaimsKey, &middleware.AuthClaims{
		User: &types.UserSessInfo{Id: "user-123"},
	}))
	rec := httptest.NewRecorder()
	handler.AuthorizeSubmit(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body StytchOAuthAuthorizeSubmitResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body.RedirectURI, "https://client.example/callback?code=abc"; got != want {
		t.Fatalf("redirect_uri = %q, want %q", got, want)
	}
	if got, want := client.submitRequest.UserID, "degov-square:user-123"; got != want {
		t.Fatalf("submit user_id = %q, want %q", got, want)
	}
	assertStringSlice(t, client.submitRequest.Scopes, []string{"openid", "degov.mcp.read"})
	assertStringSlice(t, client.submitRequest.Resources, []string{"https://square.degov.ai/mcp"})
}

func TestStytchOAuthHandlerStartUsesAuthenticatedSquareUserID(t *testing.T) {
	client := &fakeStytchOAuthClient{
		startResponse: StytchOAuthAuthorizeStartResponse{
			Client: StytchOAuthClientInfo{ClientID: "client-test"},
		},
	}
	handler := NewStytchOAuthHandler(StytchOAuthHandlerConfig{
		Client:       client,
		UserIDPrefix: "degov-square:",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/oauth/stytch/authorize/start", bytes.NewReader([]byte(`{"client_id":"client-test","redirect_uri":"https://client.example/callback"}`)))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserClaimsKey, &middleware.AuthClaims{
		User: &types.UserSessInfo{Id: "user-123"},
	}))
	rec := httptest.NewRecorder()
	handler.AuthorizeStart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got, want := client.startRequest.UserID, "degov-square:user-123"; got != want {
		t.Fatalf("start user_id = %q, want %q", got, want)
	}
}

func TestStytchOAuthHandlerReturnsGenericStartError(t *testing.T) {
	client := &fakeStytchOAuthClient{err: errors.New("project-test secret-test upstream detail")}
	handler := NewStytchOAuthHandler(StytchOAuthHandlerConfig{
		Client:       client,
		UserIDPrefix: "degov-square:",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/oauth/stytch/authorize/start", bytes.NewReader([]byte(`{"client_id":"client-test","redirect_uri":"https://client.example/callback"}`)))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserClaimsKey, &middleware.AuthClaims{
		User: &types.UserSessInfo{Id: "user-123"},
	}))
	rec := httptest.NewRecorder()
	handler.AuthorizeStart(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d, body %q", rec.Code, http.StatusBadGateway, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["error"], "Stytch authorization request failed"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestStytchOAuthHandlerReturnsGenericSubmitError(t *testing.T) {
	client := &fakeStytchOAuthClient{err: errors.New("project-test secret-test upstream detail")}
	handler := NewStytchOAuthHandler(StytchOAuthHandlerConfig{
		Client:       client,
		UserIDPrefix: "degov-square:",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/oauth/stytch/authorize/submit", bytes.NewReader([]byte(`{"client_id":"client-test","redirect_uri":"https://client.example/callback","consent_granted":true}`)))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserClaimsKey, &middleware.AuthClaims{
		User: &types.UserSessInfo{Id: "user-123"},
	}))
	rec := httptest.NewRecorder()
	handler.AuthorizeSubmit(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d, body %q", rec.Code, http.StatusBadGateway, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["error"], "Stytch authorization submit failed"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestValidateStytchOAuthDomain(t *testing.T) {
	t.Parallel()

	for _, domain := range []string{
		"https://api.stytch.com",
		"https://test.stytch.com",
		"https://splendid-pharaoh-6918.customers.stytch.com",
	} {
		if err := validateStytchOAuthDomain(domain); err != nil {
			t.Fatalf("validateStytchOAuthDomain(%q) returned error: %v", domain, err)
		}
	}

	for _, domain := range []string{
		"",
		"http://api.stytch.com",
		"https://evil.example",
		"https://api.stytch.com.evil.example",
		"https://customers.stytch.com.evil.example",
	} {
		if err := validateStytchOAuthDomain(domain); err == nil {
			t.Fatalf("validateStytchOAuthDomain(%q) returned nil, want error", domain)
		}
	}
}

func TestStytchOAuthClientRejectsUnsafeDomainBeforeRequest(t *testing.T) {
	client := NewStytchOAuthClient(StytchOAuthClientConfig{
		Domain:    "http://api.stytch.com",
		ProjectID: "project-test",
		Secret:    "secret-test",
		HTTPClient: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("HTTP client should not be called for unsafe domain")
			return nil, nil
		}).client(),
	})

	_, err := client.AuthorizeStart(context.Background(), StytchOAuthAuthorizeStartRequest{})
	if err == nil {
		t.Fatal("AuthorizeStart returned nil error, want unsafe domain error")
	}
}

type fakeStytchOAuthClient struct {
	startRequest   StytchOAuthAuthorizeStartRequest
	startResponse  StytchOAuthAuthorizeStartResponse
	submitRequest  StytchOAuthAuthorizeSubmitRequest
	submitResponse StytchOAuthAuthorizeSubmitResponse
	err            error
	startCalls     int
	submitCalls    int
}

func (c *fakeStytchOAuthClient) AuthorizeStart(_ context.Context, req StytchOAuthAuthorizeStartRequest) (StytchOAuthAuthorizeStartResponse, error) {
	c.startCalls++
	c.startRequest = req
	return c.startResponse, c.err
}

func (c *fakeStytchOAuthClient) AuthorizeSubmit(_ context.Context, req StytchOAuthAuthorizeSubmitRequest) (StytchOAuthAuthorizeSubmitResponse, error) {
	c.submitCalls++
	c.submitRequest = req
	return c.submitResponse, c.err
}

func assertJSONValue(t *testing.T, body map[string]any, key string, want any) {
	t.Helper()
	got, ok := body[key]
	if !ok {
		t.Fatalf("body missing key %q", key)
	}
	if got != want {
		t.Fatalf("body[%q] = %#v, want %#v", key, got, want)
	}
}

func assertJSONArray(t *testing.T, body map[string]any, key string, want []string) {
	t.Helper()
	value, ok := body[key]
	if !ok {
		t.Fatalf("body missing key %q", key)
	}
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("body[%q] = %#v, want JSON array", key, value)
	}
	if len(items) != len(want) {
		t.Fatalf("body[%q] length = %d, want %d", key, len(items), len(want))
	}
	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("body[%q][%d] = %#v, want %q", key, i, items[i], want[i])
		}
	}
}

func assertStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice = %v, want %v", got, want)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (f roundTripFunc) client() *http.Client {
	return &http.Client{Transport: f}
}

func allowUnsafeStytchOAuthTestDomain(t *testing.T) {
	t.Helper()
	previous := allowUnsafeStytchOAuthDomain
	allowUnsafeStytchOAuthDomain = true
	t.Cleanup(func() {
		allowUnsafeStytchOAuthDomain = previous
	})
}
