package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ringecosystem/degov-square/internal/middleware"
)

const (
	stytchOAuthResponseBodyLimit = 1 << 20
)

var allowUnsafeStytchOAuthDomain bool

type StytchOAuthClientConfig struct {
	Domain     string
	ProjectID  string
	Secret     string
	HTTPClient *http.Client
}

type StytchOAuthAuthorizer interface {
	AuthorizeStart(context.Context, StytchOAuthAuthorizeStartRequest) (StytchOAuthAuthorizeStartResponse, error)
	AuthorizeSubmit(context.Context, StytchOAuthAuthorizeSubmitRequest) (StytchOAuthAuthorizeSubmitResponse, error)
}

type StytchOAuthClient struct {
	cfg        StytchOAuthClientConfig
	httpClient *http.Client
}

type StytchOAuthAuthorizeRequest struct {
	ClientID            string   `json:"client_id"`
	RedirectURI         string   `json:"redirect_uri"`
	ResponseType        string   `json:"response_type"`
	Scopes              []string `json:"scopes"`
	UserID              string   `json:"user_id,omitempty"`
	State               string   `json:"state,omitempty"`
	Nonce               string   `json:"nonce,omitempty"`
	CodeChallenge       string   `json:"code_challenge,omitempty"`
	CodeChallengeMethod string   `json:"code_challenge_method,omitempty"`
	Resources           []string `json:"resources,omitempty"`
}

type StytchOAuthAuthorizeStartRequest struct {
	StytchOAuthAuthorizeRequest
}

type StytchOAuthAuthorizeSubmitRequest struct {
	StytchOAuthAuthorizeRequest
	ConsentGranted bool `json:"consent_granted"`
}

type StytchOAuthAuthorizeStartResponse struct {
	Client          StytchOAuthClientInfo    `json:"client"`
	ConsentRequired bool                     `json:"consent_required"`
	ScopeResults    []StytchOAuthScopeResult `json:"scope_results"`
}

type StytchOAuthClientInfo struct {
	ClientID          string `json:"client_id"`
	ClientName        string `json:"client_name"`
	ClientDescription string `json:"client_description"`
	ClientType        string `json:"client_type"`
	LogoURL           string `json:"logo_url"`
}

type StytchOAuthScopeResult struct {
	Scope       string `json:"scope"`
	Description string `json:"description"`
	IsGrantable bool   `json:"is_grantable"`
}

type StytchOAuthAuthorizeSubmitResponse struct {
	RedirectURI string `json:"redirect_uri"`
}

type StytchOAuthHandlerConfig struct {
	Client        StytchOAuthAuthorizer
	UserIDPrefix  string
	OAuthResource string
}

type StytchOAuthHandler struct {
	cfg StytchOAuthHandlerConfig
}

type stytchOAuthWebRequest struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	ResponseType        string `json:"response_type"`
	Scope               string `json:"scope"`
	State               string `json:"state"`
	Nonce               string `json:"nonce"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	ConsentGranted      bool   `json:"consent_granted"`
}

func NewStytchOAuthClient(cfg StytchOAuthClientConfig) *StytchOAuthClient {
	cfg.Domain = strings.TrimRight(cfg.Domain, "/")
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &StytchOAuthClient{cfg: cfg, httpClient: client}
}

func (c *StytchOAuthClient) AuthorizeStart(ctx context.Context, req StytchOAuthAuthorizeStartRequest) (StytchOAuthAuthorizeStartResponse, error) {
	var resp StytchOAuthAuthorizeStartResponse
	err := c.post(ctx, c.authorizeStartPath(), req.payload(), &resp)
	return resp, err
}

func (c *StytchOAuthClient) AuthorizeSubmit(ctx context.Context, req StytchOAuthAuthorizeSubmitRequest) (StytchOAuthAuthorizeSubmitResponse, error) {
	var resp StytchOAuthAuthorizeSubmitResponse
	err := c.post(ctx, c.authorizeSubmitPath(), req.payload(), &resp)
	return resp, err
}

func (r StytchOAuthAuthorizeStartRequest) payload() StytchOAuthAuthorizeRequest {
	req := r.StytchOAuthAuthorizeRequest
	return StytchOAuthAuthorizeRequest{
		ClientID:     req.ClientID,
		RedirectURI:  req.RedirectURI,
		ResponseType: req.ResponseType,
		Scopes:       append([]string(nil), req.Scopes...),
		UserID:       req.UserID,
	}
}

func (r StytchOAuthAuthorizeSubmitRequest) payload() StytchOAuthAuthorizeSubmitRequest {
	req := r.StytchOAuthAuthorizeRequest
	return StytchOAuthAuthorizeSubmitRequest{
		StytchOAuthAuthorizeRequest: StytchOAuthAuthorizeRequest{
			ClientID:      req.ClientID,
			RedirectURI:   req.RedirectURI,
			ResponseType:  req.ResponseType,
			Scopes:        append([]string(nil), req.Scopes...),
			UserID:        req.UserID,
			State:         req.State,
			Nonce:         req.Nonce,
			CodeChallenge: req.CodeChallenge,
			Resources:     append([]string(nil), req.Resources...),
		},
		ConsentGranted: r.ConsentGranted,
	}
}

func (c *StytchOAuthClient) authorizeStartPath() string {
	return "/v1/idp/oauth/authorize/start"
}

func (c *StytchOAuthClient) authorizeSubmitPath() string {
	return "/v1/idp/oauth/authorize"
}

func (c *StytchOAuthClient) post(ctx context.Context, path string, payload any, target any) error {
	if c.cfg.Domain == "" {
		return errors.New("missing Stytch OAuth domain")
	}
	if !allowUnsafeStytchOAuthDomain {
		if err := validateStytchOAuthDomain(c.cfg.Domain); err != nil {
			return err
		}
	}
	if c.cfg.ProjectID == "" {
		return errors.New("missing Stytch OAuth project ID")
	}
	if c.cfg.Secret == "" {
		return errors.New("missing Stytch OAuth secret")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Domain+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.cfg.ProjectID, c.cfg.Secret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := readLimitedResponseBody(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Stytch OAuth request returned status %d: %s", resp.StatusCode, cleanStytchError(respBody))
	}
	if err := json.Unmarshal(respBody, target); err != nil {
		return fmt.Errorf("decode Stytch OAuth response: %w", err)
	}
	return nil
}

func validateStytchOAuthDomain(domain string) error {
	parsed, err := url.Parse(domain)
	if err != nil {
		return fmt.Errorf("invalid Stytch OAuth domain: %w", err)
	}
	if parsed.Scheme != "https" {
		return errors.New("Stytch OAuth domain must use https")
	}
	hostname := strings.ToLower(parsed.Hostname())
	switch hostname {
	case "api.stytch.com", "test.stytch.com":
		return nil
	default:
		if strings.HasSuffix(hostname, ".customers.stytch.com") {
			return nil
		}
		return errors.New("Stytch OAuth domain host is not allowed")
	}
}

func readLimitedResponseBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, stytchOAuthResponseBodyLimit+1)
	respBody, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(respBody) > stytchOAuthResponseBodyLimit {
		return nil, errors.New("Stytch OAuth response body too large")
	}
	return respBody, nil
}

func cleanStytchError(body []byte) string {
	var parsed struct {
		ErrorMessage string `json:"error_message"`
		ErrorType    string `json:"error_type"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		switch {
		case parsed.ErrorMessage != "":
			return parsed.ErrorMessage
		case parsed.ErrorType != "":
			return parsed.ErrorType
		}
	}
	message := strings.TrimSpace(string(body))
	if message == "" {
		return "empty response"
	}
	if len(message) > 300 {
		return message[:300]
	}
	return message
}

func NewStytchOAuthHandler(cfg StytchOAuthHandlerConfig) *StytchOAuthHandler {
	if cfg.UserIDPrefix == "" {
		cfg.UserIDPrefix = "degov-square:"
	}
	return &StytchOAuthHandler{cfg: cfg}
}

func (h *StytchOAuthHandler) AuthorizeStart(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.requireAuth(w, r)
	if !ok {
		return
	}

	var webReq stytchOAuthWebRequest
	if !decodeJSONRequest(w, r, &webReq) {
		return
	}

	req := h.buildAuthorizeRequest(webReq, claims)
	resp, err := h.cfg.Client.AuthorizeStart(r.Context(), StytchOAuthAuthorizeStartRequest{
		StytchOAuthAuthorizeRequest: req,
	})
	if err != nil {
		slog.Error("Stytch OAuth authorize start failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, "Stytch authorization request failed")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *StytchOAuthHandler) AuthorizeSubmit(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.requireAuth(w, r)
	if !ok {
		return
	}

	var webReq stytchOAuthWebRequest
	if !decodeJSONRequest(w, r, &webReq) {
		return
	}

	req := h.buildAuthorizeRequest(webReq, claims)
	if h.cfg.OAuthResource != "" {
		req.Resources = []string{h.cfg.OAuthResource}
	}
	resp, err := h.cfg.Client.AuthorizeSubmit(r.Context(), StytchOAuthAuthorizeSubmitRequest{
		StytchOAuthAuthorizeRequest: req,
		ConsentGranted:              webReq.ConsentGranted,
	})
	if err != nil {
		slog.Error("Stytch OAuth authorize submit failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, "Stytch authorization submit failed")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *StytchOAuthHandler) requireAuth(w http.ResponseWriter, r *http.Request) (*middleware.AuthClaims, bool) {
	claims, err := middleware.RequireAuth(r.Context())
	if err != nil || claims.User == nil || claims.User.Id == "" {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return nil, false
	}
	return claims, true
}

func (h *StytchOAuthHandler) buildAuthorizeRequest(webReq stytchOAuthWebRequest, claims *middleware.AuthClaims) StytchOAuthAuthorizeRequest {
	responseType := strings.TrimSpace(webReq.ResponseType)
	if responseType == "" {
		responseType = "code"
	}
	req := StytchOAuthAuthorizeRequest{
		ClientID:            webReq.ClientID,
		RedirectURI:         webReq.RedirectURI,
		ResponseType:        responseType,
		Scopes:              strings.Fields(webReq.Scope),
		State:               webReq.State,
		Nonce:               webReq.Nonce,
		CodeChallenge:       webReq.CodeChallenge,
		CodeChallengeMethod: webReq.CodeChallengeMethod,
	}

	req.UserID = h.cfg.UserIDPrefix + claims.User.Id
	return req
}

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, stytchOAuthResponseBodyLimit)).Decode(target); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
