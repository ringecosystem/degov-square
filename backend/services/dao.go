package services

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/dbmodels"
	"github.com/ringecosystem/degov-apps/graph/model"
	"github.com/ringecosystem/degov-apps/internal"
	"github.com/ringecosystem/degov-apps/internal/database"
	"github.com/ringecosystem/degov-apps/internal/utils"
	"github.com/ringecosystem/degov-apps/types"
)

type DaoService struct {
	db *gorm.DB
}

func NewDaoService() *DaoService {
	return &DaoService{
		db: database.GetDB(),
	}
}

func (s *DaoService) GetDaos() ([]*model.Dao, error) {
	var dbDaos []dbmodels.Dao
	if err := s.db.Table("dgv_dao").Where("state = ?", "ACTIVE").Find(&dbDaos).Order("seq asc").Error; err != nil {
		return nil, err
	}

	// Extract dao codes from dbDaos
	var daoCodes []string
	for _, dao := range dbDaos {
		daoCodes = append(daoCodes, dao.Code)
	}

	var dbChips []dbmodels.DgvDaoChip
	if err := s.db.Table("dgv_dao_chip").Where("dao_code IN ?", daoCodes).Find(&dbChips).Error; err != nil {
		return nil, err
	}

	var daos []*model.Dao
	for _, dbDao := range dbDaos {
		liked := false
		subscribed := false

		// Convert tags from JSON string to string array
		var tags []string
		if dbDao.Tags != "" {
			if err := json.Unmarshal([]byte(dbDao.Tags), &tags); err != nil {
				// If JSON unmarshal fails, treat as empty array
				tags = []string{}
			}
		}

		// Find chips for this DAO
		var chips []*model.DaoChip
		for _, dbChip := range dbChips {
			if dbChip.DaoCode == dbDao.Code {
				chip := &model.DaoChip{
					ID:         dbChip.ID,
					DaoCode:    dbChip.DaoCode,
					ChipCode:   dbChip.ChipCode,
					Value:      dbChip.Value,
					Additional: &dbChip.Additional,
					Ctime:      dbChip.CTime,
					Utime:      dbChip.UTime,
				}
				chips = append(chips, chip)
			}
		}

		dao := &model.Dao{
			ID:         dbDao.ID,
			ChainID:    int32(dbDao.ChainID),
			ChainName:  dbDao.ChainName,
			Name:       dbDao.Name,
			Code:       dbDao.Code,
			Seq:        int32(dbDao.Seq),
			State:      dbDao.State,
			Tags:       tags,
			TimeSyncd:  dbDao.TimeSyncd,
			Ctime:      dbDao.CTime,
			Utime:      dbDao.UTime,
			Liked:      &liked,
			Subscribed: &subscribed,
			Chips:      chips,
		}

		daos = append(daos, dao)
	}

	return daos, nil
}

func (s *DaoService) RefreshDaoAndConfig(input types.RefreshDaoAndConfigInput) error {
	var existingDao dbmodels.Dao
	result := s.db.Where("code = ?", input.Code).First(&existingDao)

	tagsJson := utils.ToJSON(input.Tags)
	if result.Error == gorm.ErrRecordNotFound {
		// Insert new DAO
		dao := &dbmodels.Dao{
			ID:         internal.NextIDString(),
			ChainID:    input.Config.Chain.ID,
			ChainName:  input.Config.Chain.Name,
			Name:       input.Config.Name,
			Code:       input.Code,
			State:      "ACTIVE",
			Tags:       tagsJson,
			ConfigLink: input.ConfigLink,
			TimeSyncd:  utils.TimePtrNow(),
		}
		if err := s.db.Create(dao).Error; err != nil {
			return err
		}
	} else {
		// Update existing DAO
		existingDao.ChainID = input.Config.Chain.ID
		existingDao.ChainName = input.Config.Chain.Name
		existingDao.Name = input.Config.Name
		existingDao.State = "ACTIVE"
		existingDao.Tags = tagsJson
		existingDao.ConfigLink = input.ConfigLink
		existingDao.UTime = utils.TimePtrNow()
		if err := s.db.Save(&existingDao).Error; err != nil {
			return err
		}
	}

	var existingConfig dbmodels.DgvDaoConfig
	r2 := s.db.Where("dao_code = ?", input.Code).First(&existingConfig)

	if r2.Error == gorm.ErrRecordNotFound {
		// Insert new DAO config
		config := &dbmodels.DgvDaoConfig{
			ID:      internal.NextIDString(),
			DaoCode: input.Code,
			Config:  input.Raw,
			CTime:   time.Now(),
		}
		return s.db.Create(config).Error
	} else if r2.Error != nil {
		return r2.Error
	}

	// Update existing DAO config
	existingConfig.Config = input.Raw
	existingConfig.UTime = utils.TimePtrNow()

	return s.db.Save(&existingConfig).Error
}

// MarkInactiveDAOs marks DAOs as inactive if they're not in the active list
func (s *DaoService) MarkInactiveDAOs(activeCodes map[string]bool) error {
	// Use a more efficient query to find and update inactive DAOs in one go
	result := s.db.Model(&dbmodels.Dao{}).
		Where("code NOT IN ? AND state != ?", getMapKeys(activeCodes), "INACTIVE").
		Updates(map[string]interface{}{
			"state": "INACTIVE",
			"utime": utils.TimePtrNow(),
		})

	if result.Error != nil {
		return result.Error
	}

	return nil
}

// getMapKeys extracts keys from a map[string]bool
func getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

type DaoChipService struct {
	db *gorm.DB
}

func NewDaoChipService() *DaoChipService {
	return &DaoChipService{
		db: database.GetDB(),
	}
}

func (s *DaoChipService) StoreChipAgent(input types.StoreDaoChipInput) error {
	chipCode := "AGENT"
	var existingChip dbmodels.DgvDaoChip
	result := s.db.Where("dao_code = ? AND chip_code = ?", input.Code, chipCode).First(&existingChip)
	if result.Error == gorm.ErrRecordNotFound {
		// Insert new chip
		chip := &dbmodels.DgvDaoChip{
			ID:         internal.NextIDString(),
			DaoCode:    input.Code,
			ChipCode:   chipCode,
			Value:      "ENABLED",
			Additional: utils.ToJSON(input.AgentConfig),
			CTime:      time.Now(),
		}
		if err := s.db.Create(chip).Error; err != nil {
			return err
		}
		return nil
	}
	if result.Error != nil {
		return result.Error
	}
	existingChip.Value = "ENABLED"
	existingChip.Additional = utils.ToJSON(input.AgentConfig)
	existingChip.UTime = utils.TimePtrNow()
	if err := s.db.Save(&existingChip).Error; err != nil {
		return err
	}
	return nil
}
