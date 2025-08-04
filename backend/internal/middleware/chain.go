package middleware

import (
	"net/http"
)

// Middleware represents a middleware function
type Middleware func(http.Handler) http.Handler

// Chain represents a chain of middlewares
type Chain struct {
	middlewares []Middleware
}

// NewChain creates a new middleware chain
func NewChain(middlewares ...Middleware) *Chain {
	return &Chain{
		middlewares: middlewares,
	}
}

// Use adds middleware to the chain
func (c *Chain) Use(middleware Middleware) *Chain {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

// Then applies all middlewares to the final handler
func (c *Chain) Then(finalHandler http.Handler) http.Handler {
	// Apply middlewares in reverse order so they execute in the correct order
	handler := finalHandler
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

// ThenFunc is a convenience method that accepts http.HandlerFunc
func (c *Chain) ThenFunc(fn http.HandlerFunc) http.Handler {
	return c.Then(fn)
}
