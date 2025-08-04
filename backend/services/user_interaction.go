package services

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/internal"
	"github.com/ringecosystem/degov-apps/types"
)

type UserInteractionService struct {
	db         *gorm.DB
	daoService *DaoService
}

func NewUserInteractionService() *UserInteractionService {
	return &UserInteractionService{
		db:         database.GetDB(),
		daoService: NewDaoService(),
	}
}

func (s *UserInteractionService) ModifyLikeDao(baseInput types.BasicInput[gqlmodels.ModifyLikeDaoInput]) (bool, error) {
	usess := baseInput.User
	input := baseInput.Input

	_, err := s.daoService.Inspect(input.DaoCode)
	if err != nil {
		return false, fmt.Errorf("error inspecting DAO: %w", err)
	}

	var existingLikedDao dbmodels.UserLikedDao
	err = s.db.Where("user_id = ? AND dao_code = ?", usess.Id, input.DaoCode).First(&existingLikedDao).Error
	isNotFoundError := false
	if err != nil {
		isNotFoundError = errors.Is(err, gorm.ErrRecordNotFound)
		if !isNotFoundError {
			return false, fmt.Errorf("error checking existing like: %w", err)
		}
	}

	switch baseInput.Input.Action {
	case gqlmodels.LikeActionLike:
		if isNotFoundError {
			// Create new like
			like := &dbmodels.UserLikedDao{
				ID:          internal.NextIDString(),
				DaoCode:     input.DaoCode,
				UserID:      usess.Id,
				UserAddress: usess.Address,
				CTime:       time.Now(),
			}

			if err := s.db.Create(like).Error; err != nil {
				return false, fmt.Errorf("error creating like: %w", err)
			}
			return true, nil
		}
	case gqlmodels.LikeActionUnlike:
		if !isNotFoundError {
			// Remove existing like
			result := s.db.Where("user_id = ? AND dao_code = ?", usess.Id, input.DaoCode).Delete(&dbmodels.UserLikedDao{})
			if result.Error != nil {
				return false, fmt.Errorf("error removing like: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				return false, errors.New("like not found")
			}
			return true, nil
		}
	}

	return true, nil
}

// UserLikedDao methods
func (s *UserInteractionService) LikeDao(userID, daoCode string) (*dbmodels.UserLikedDao, error) {
	// check if already liked
	var existing dbmodels.UserLikedDao
	err := s.db.Where("user_id = ? AND dao_code = ?", userID, daoCode).First(&existing).Error
	if err == nil {
		return nil, errors.New("DAO already liked by user")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error checking existing like: %w", err)
	}

	// generate like ID
	likeID := fmt.Sprintf("like_%d", "hello")

	like := &dbmodels.UserLikedDao{
		ID:      likeID,
		DaoCode: daoCode,
		UserID:  userID,
		CTime:   time.Now(),
	}

	if err := s.db.Create(like).Error; err != nil {
		return nil, fmt.Errorf("error creating like: %w", err)
	}

	return like, nil
}

func (s *UserInteractionService) UnlikeDao(userID, daoCode string) error {
	result := s.db.Where("user_id = ? AND dao_code = ?", userID, daoCode).Delete(&dbmodels.UserLikedDao{})
	if result.Error != nil {
		return fmt.Errorf("error removing like: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("like not found")
	}
	return nil
}
