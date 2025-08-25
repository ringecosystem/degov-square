package services

import (
	"github.com/ringecosystem/degov-apps/database"
	"gorm.io/gorm"
)

type EvmChainService struct {
	db *gorm.DB
}

func NewEvmChainService() *EvmChainService {
	return &EvmChainService{
		db: database.GetDB(),
	}
}
