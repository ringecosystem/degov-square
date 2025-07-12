package services

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/internal/database"
	"github.com/ringecosystem/degov-apps/models"
)

type DaoService struct {
	db *gorm.DB
}

func NewDaoService() *DaoService {
	return &DaoService{
		db: database.GetDB(),
	}
}

func (s *DaoService) CreateDao(chainID int, chainName, name, code, configLink string) (*models.Dao, error) {
	// check if code already exists
	var existingDao models.Dao
	err := s.db.Where("code = ?", code).First(&existingDao).Error
	if err == nil {
		return nil, errors.New("DAO code already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error checking existing DAO: %w", err)
	}

	// generate dao ID
	daoID := fmt.Sprintf("dao_%d", s.generateDaoID())

	dao := &models.Dao{
		ID:         daoID,
		ChainID:    chainID,
		ChainName:  chainName,
		Name:       name,
		Code:       code,
		ConfigLink: configLink,
		CTime:      time.Now(),
	}

	if err := s.db.Create(dao).Error; err != nil {
		return nil, fmt.Errorf("error creating DAO: %w", err)
	}

	return dao, nil
}

func (s *DaoService) GetDaoByID(id string) (*models.Dao, error) {
	var dao models.Dao
	err := s.db.First(&dao, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("DAO not found")
		}
		return nil, fmt.Errorf("error finding DAO: %w", err)
	}
	return &dao, nil
}

func (s *DaoService) GetDaoByCode(code string) (*models.Dao, error) {
	var dao models.Dao
	err := s.db.Where("code = ?", code).First(&dao).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("DAO not found")
		}
		return nil, fmt.Errorf("error finding DAO: %w", err)
	}
	return &dao, nil
}

func (s *DaoService) UpdateDao(id string, chainID int, chainName, name, code, configLink string) (*models.Dao, error) {
	var dao models.Dao
	err := s.db.First(&dao, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("DAO not found")
		}
		return nil, fmt.Errorf("error finding DAO: %w", err)
	}

	// check if new code conflicts with existing DAO (if code is being changed)
	if dao.Code != code {
		var existingDao models.Dao
		err := s.db.Where("code = ? AND id != ?", code, id).First(&existingDao).Error
		if err == nil {
			return nil, errors.New("DAO code already exists")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("error checking existing DAO: %w", err)
		}
	}

	dao.ChainID = chainID
	dao.ChainName = chainName
	dao.Name = name
	dao.Code = code
	dao.ConfigLink = configLink

	if err := s.db.Save(&dao).Error; err != nil {
		return nil, fmt.Errorf("error updating DAO: %w", err)
	}

	return &dao, nil
}

func (s *DaoService) DeleteDao(id string) error {
	result := s.db.Delete(&models.Dao{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("error deleting DAO: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("DAO not found")
	}
	return nil
}

func (s *DaoService) GetDaos() ([]*models.Dao, error) {
	var daos []*models.Dao
	err := s.db.Find(&daos).Error
	if err != nil {
		return nil, fmt.Errorf("error getting DAOs: %w", err)
	}
	return daos, nil
}

func (s *DaoService) UpdateSyncTime(id string, syncTime time.Time) error {
	result := s.db.Model(&models.Dao{}).Where("id = ?", id).Update("time_sync", syncTime)
	if result.Error != nil {
		return fmt.Errorf("error updating sync time: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("DAO not found")
	}
	return nil
}

func (s *DaoService) generateDaoID() int64 {
	// Simple implementation - in production, you'd want to use UUID or a more robust ID generation
	var count int64
	s.db.Model(&models.Dao{}).Count(&count)
	return count + 1
}
