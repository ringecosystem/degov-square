package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Capture response status
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			remoteAddr := r.RemoteAddr
			if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
				remoteAddr = realIP
			} else if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
				remoteAddr = forwardedFor
			}

			// Get content length from response
			contentLength := wrapped.ResponseWriter.Header().Get("Content-Length")
			if contentLength == "" {
				contentLength = "-"
			}

			userAgent := r.UserAgent()
			if userAgent == "" {
				userAgent = "-"
			}

			// Format: METHOD /path?query HTTP/1.1 IP - status size "user-agent" duration request_id
			logMsg := fmt.Sprintf(`%s %s %s %s - %d %s "%s" %s`,
				r.Method,
				r.RequestURI,
				r.Proto,
				remoteAddr,
				wrapped.statusCode,
				contentLength,
				userAgent,
				duration.String(),
			)

			slog.Info(logMsg)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					slog.Error("Panic recovered",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	}
}
