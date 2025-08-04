//go:generate go run github.com/99designs/gqlgen generate

package graph

import (
	"github.com/ringecosystem/degov-apps/internal/middleware"
	"github.com/ringecosystem/degov-apps/services"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	authUtils *middleware.AuthUtils

	authService            *services.AuthService
	daoService             *services.DaoService
	daoConfigService       *services.DaoConfigService
	userInteractionService *services.UserInteractionService
}

func NewResolver() *Resolver {
	return &Resolver{
		authUtils: middleware.NewAuthUtils(),

		authService:            services.NewAuthService(),
		daoService:             services.NewDaoService(),
		daoConfigService:       services.NewDaoConfigService(),
		userInteractionService: services.NewUserInteractionService(),
	}
}
