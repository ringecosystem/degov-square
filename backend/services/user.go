package services

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/dbmodels"
	"github.com/ringecosystem/degov-apps/internal/database"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService() *UserService {
	return &UserService{
		db: database.GetDB(),
	}
}

func (s *UserService) Nonce() (string, error) {
	// Generate a nonce for user authentication
	// This is a placeholder implementation; actual nonce generation logic should be added
	return "nonce-placeholder", nil
}

func (s *UserService) CreateUser(address string, email *string) (*dbmodels.User, error) {
	// check if address already exists
	var existingUser dbmodels.User
	err := s.db.Where("address = ?", address).First(&existingUser).Error
	if err == nil {
		return nil, errors.New("address already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error checking existing user: %w", err)
	}

	// generate user ID (you might want to use UUID here)
	userID := fmt.Sprintf("user_%d", s.generateUserID())

	user := &dbmodels.User{
		ID:      userID,
		Address: address,
		Email:   email,
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	return user, nil
}

func (s *UserService) GetUserByID(id string) (*dbmodels.User, error) {
	var user dbmodels.User
	err := s.db.First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	return &user, nil
}

func (s *UserService) GetUserByAddress(address string) (*dbmodels.User, error) {
	var user dbmodels.User
	err := s.db.Where("address = ?", address).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	return &user, nil
}

func (s *UserService) UpdateUser(id string, email *string) (*dbmodels.User, error) {
	var user dbmodels.User
	err := s.db.First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("error finding user: %w", err)
	}

	user.Email = email

	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("error updating user: %w", err)
	}

	return &user, nil
}

func (s *UserService) DeleteUser(id string) error {
	result := s.db.Delete(&dbmodels.User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("error deleting user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (s *UserService) GetUsers() ([]*dbmodels.User, error) {
	var users []*dbmodels.User
	err := s.db.Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("error getting users: %w", err)
	}
	return users, nil
}

func (s *UserService) generateUserID() int64 {
	// Simple implementation - in production, you'd want to use UUID or a more robust ID generation
	var count int64
	s.db.Model(&dbmodels.User{}).Count(&count)
	return count + 1
}
