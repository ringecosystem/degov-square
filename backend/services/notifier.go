package services

import (
	"github.com/ringecosystem/degov-apps/types"
	"gorm.io/gorm"
)

var globalNotifier *NotifierService

func getNotifier() *NotifierService {
	if globalNotifier == nil {
		globalNotifier = &NotifierService{}
	}
	return globalNotifier
}

type NotifierService struct {
	db *gorm.DB
}

func NewNotifierService() *NotifierService {
	return getNotifier()
}

func (n *NotifierService) Notify(input types.NotifyInput) error {
	return nil
}
