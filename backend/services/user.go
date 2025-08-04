package services

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	"github.com/ringecosystem/degov-apps/internal/utils"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService() *UserService {
	return &UserService{
		db: database.GetDB(),
	}
}

func (s *UserService) Modify(input dbmodels.User) (*dbmodels.User, error) {
	address := strings.ToLower(input.Address)
	// check if address already exists
	var existingUser dbmodels.User
	err := s.db.Where("address = ?", address).First(&existingUser).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Address does not exist, create new user
		user := &dbmodels.User{
			ID:      utils.NextIDString(),
			Address: address,
			Email:   input.Email,
		}

		if err := s.db.Create(user).Error; err != nil {
			return nil, fmt.Errorf("error creating user: %w", err)
		}

		return user, nil
	}

	if existingUser.Email != nil && input.Email != nil && *existingUser.Email != *input.Email {
		existingUser.Email = input.Email
		if err := s.db.Save(&existingUser).Error; err != nil {
			return nil, fmt.Errorf("error updating user: %w", err)
		}
	}
	return &existingUser, nil
}

func (s *UserService) Inspect(seed string) (*dbmodels.User, error) {
	var user dbmodels.User
	err := s.db.Where("address = ?", seed).Or("id = ?", seed).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	return &user, nil
}
