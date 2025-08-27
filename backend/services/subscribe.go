package services

import (
	"github.com/ringecosystem/degov-apps/database"
	"gorm.io/gorm"
)

type SubscribeService struct {
	db         *gorm.DB
}

func NewSubscribeService() *SubscribeService {
	return &SubscribeService{
		db: database.GetDB(),
	}
}
