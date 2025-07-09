package graph

import (
	"github.com/ringecosystem/degov-apps/services"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	userService *services.UserService
}

func NewResolver() *Resolver {
	return &Resolver{
		userService: services.NewUserService(),
	}
}
