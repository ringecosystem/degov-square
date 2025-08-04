package services

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/types"
)

type UserInteractionService struct {
	db *gorm.DB
}

func NewUserInteractionService() *UserInteractionService {
	return &UserInteractionService{
		db: database.GetDB(),
	}
}

func (s *UserInteractionService) ModifyLikeDao(baseInput types.BasicInput[gqlmodels.ModifyLikeDaoInput]) (bool, error) {
	// userID := baseInput.User.ID
	// daoCode := baseInput.Input.DaoCode
	// action := baseInput.Input.Action

	// if action == types.LikeActionLike {
	// 	// Like the DAO
	// 	_, err := s.LikeDao(userID, daoCode)
	// 	if err != nil {
	// 		return false, fmt.Errorf("error liking DAO: %w", err)
	// 	}
	// 	return true, nil
	// } else if action == types.LikeActionUnlike {
	// 	// Unlike the DAO
	// 	err := s.UnlikeDao(userID, daoCode)
	// 	if err != nil {
	// 		return false, fmt.Errorf("error unliking DAO: %w", err)
	// 	}
	// 	return true, nil
	// }
	// return false, errors.New("invalid action")
	return false, nil
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
