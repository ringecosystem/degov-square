package services

import (
	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/database"
	"github.com/ringecosystem/degov-apps/types"
)

type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		db: database.GetDB(),
	}
}

func (s *NotificationService) StoreVoteNotification(input *types.StoreVoteNotificationInput) error {
	// return s.db.Create(notification).Error

	/*

		select * from dgv_subscribed_feature as f
		where feature='ENABLE_VOTED' and strategy='enable' and dao_code='x'

	*/

	return nil
}
