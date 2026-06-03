package mcp

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const (
	AuthModeBearer = "bearer"
	AuthModeNone   = "none"
	AuthModeOAuth  = "oauth"
)

func BearerAuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !validBearerToken(r, token) {
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func validBearerToken(r *http.Request, token string) bool {
	if token == "" {
		return false
	}

	got := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(got) <= len(prefix) || got[:len(prefix)] != prefix {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(got[len(prefix):]), []byte(token)) == 1
}

type authModes struct {
	bearer bool
	none   bool
	oauth  bool
	valid  bool
}

func parseAuthModes(authMode string) authModes {
	modes := authModes{valid: true}
	parts := strings.Split(authMode, ",")
	for _, part := range parts {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "":
			modes.valid = false
		case AuthModeBearer:
			modes.bearer = true
		case AuthModeNone:
			modes.none = true
		case AuthModeOAuth:
			modes.oauth = true
		default:
			modes.valid = false
		}
	}
	if modes.none && (modes.bearer || modes.oauth) {
		modes.valid = false
	}
	if !modes.bearer && !modes.none && !modes.oauth {
		modes.valid = false
	}
	return modes
}

func AuthModeIncludes(authMode string, mode string) bool {
	modes := parseAuthModes(authMode)
	if !modes.valid {
		return false
	}
	switch mode {
	case AuthModeBearer:
		return modes.bearer
	case AuthModeNone:
		return modes.none
	case AuthModeOAuth:
		return modes.oauth
	default:
		return false
	}
}
