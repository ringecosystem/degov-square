package utils

import (
	"context"
	"fmt"

	"github.com/ringecosystem/degov-apps/internal/middleware"
	"github.com/ringecosystem/degov-apps/types"
)

// AuthUtils provides utility functions for authentication and authorization
type AuthUtils struct{}

// NewAuthUtils creates a new AuthUtils instance
func NewAuthUtils() *AuthUtils {
	return &AuthUtils{}
}

// GetUser extracts the user information from the authentication context
func (a *AuthUtils) GetUser(ctx context.Context) (*types.UserSessInfo, error) {
	claims, err := a.GetAuthClaims(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %v", err)
	}
	return claims.User, nil
}

// // GetOptionalUserAddress extracts the user address if authenticated, returns empty string if not
// func (a *AuthUtils) GetOptionalUserAddress(ctx context.Context) string {
// 	claims, ok := middleware.GetUserFromContext(ctx)
// 	if !ok || claims == nil {
// 		return ""
// 	}
// 	return claims.Address
// }

// // IsUserAuthorizedForResource checks if the authenticated user is authorized to access a resource
// func (a *AuthUtils) IsUserAuthorizedForResource(ctx context.Context, resourceOwnerID string) error {
// 	claims, err := middleware.RequireAuth(ctx)
// 	if err != nil {
// 		return fmt.Errorf("authentication required: %v", err)
// 	}

// 	if claims.Address != resourceOwnerID {
// 		return fmt.Errorf("permission denied: insufficient privileges to access this resource")
// 	}

// 	return nil
// }

// // RequireAuthWithOwnership ensures user is authenticated and authorized for a specific resource
// func (a *AuthUtils) RequireAuthWithOwnership(ctx context.Context, resourceOwnerID string) (string, error) {
// 	claims, err := middleware.RequireAuth(ctx)
// 	if err != nil {
// 		return "", fmt.Errorf("authentication required: %v", err)
// 	}

// 	if claims.Address != resourceOwnerID {
// 		return "", fmt.Errorf("permission denied: insufficient privileges to access this resource")
// 	}

// 	return claims.Address, nil
// }

// GetAuthClaims extracts full user claims from context
func (a *AuthUtils) GetAuthClaims(ctx context.Context) (*middleware.AuthClaims, error) {
	return middleware.RequireAuth(ctx)
}

// IsAuthenticated checks if request has valid authentication
func (a *AuthUtils) IsAuthenticated(ctx context.Context) bool {
	return middleware.IsAuthenticated(ctx)
}
