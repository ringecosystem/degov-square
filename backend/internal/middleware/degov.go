package middleware

import (
	"net/http"

	"github.com/ringecosystem/degov-apps/services"
)

type DegovMiddleware struct {
	daoService *services.DaoService // Assuming DaoService is defined elsewhere
}

func NewDegovMiddleware() *DegovMiddleware {
	return &DegovMiddleware{
		daoService: services.NewDaoService(),
	}
}

// Middleware returns a standard middleware function
func (m *DegovMiddleware) Middleware() Middleware {
	return func(next http.Handler) http.Handler {
		return m.HTTPMiddleware(next)
	}
}

func (m *DegovMiddleware) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		next.ServeHTTP(w, r)
	})
}
