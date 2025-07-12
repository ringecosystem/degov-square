package services

import (
	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/internal/database"
)

type DaoService struct {
	db *gorm.DB
}

func NewDaoService() *DaoService {
	return &DaoService{
		db: database.GetDB(),
	}
}
