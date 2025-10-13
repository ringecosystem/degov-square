package services

import (
	"github.com/ringecosystem/degov-square/database"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"gorm.io/gorm"
)

type TreasuryService struct {
	db *gorm.DB
}

func NewTreasuryService() *TreasuryService {
	return &TreasuryService{
		db: database.GetDB(),
	}
}

// Load all assets of treasury
func (s *TreasuryService) LoadTreasuryAssets(input *gqlmodels.TreasuryAssetsInput) ([]*gqlmodels.TreasuryAsset, error) {
	var assets []*gqlmodels.TreasuryAsset
	// err := s.db.Where("chain = ? AND address = ?", chain, address).Find(&assets).Error
	// if err != nil {
	// 	return nil, err
	// }
	return assets, nil
}
