package mcp

import (
	"crypto/subtle"
	"net/http"
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
