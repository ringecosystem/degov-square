//go:generate go run github.com/99designs/gqlgen generate

package graph

import (
	"github.com/ringecosystem/degov-apps/services"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	authService *services.AuthService
	// userService            *services.UserService
	daoService *services.DaoService
	// userInteractionService *services.UserInteractionService
	// notificationService    *services.NotificationService
}

func NewResolver() *Resolver {
	return &Resolver{
		authService: services.NewAuthService(),
		// userService:            services.NewUserService(),
		daoService: services.NewDaoService(),
		// userInteractionService: services.NewUserInteractionService(),
		// notificationService:    services.NewNotificationService(),
	}
}
