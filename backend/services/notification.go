package services

import (
	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	"github.com/ringecosystem/degov-apps/internal/utils"
)

type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		db: database.GetDB(),
	}
}

func (s *NotificationService) StoreRecords(records []dbmodels.NotificationRecord) error {
	if len(records) == 0 {
		return nil
	}

	for i := range records {
		records[i].ID = utils.NextIDString()
	}

	if err := s.db.Create(&records).Error; err != nil {
		return err
	}

	return nil
}
