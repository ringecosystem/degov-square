package services

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/internal"
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

func (s *DaoService) convertToGqlDao(dbDao dbmodels.Dao) *gqlmodels.Dao {
	var tags []string
	if dbDao.Tags != "" {
		if err := json.Unmarshal([]byte(dbDao.Tags), &tags); err != nil {
			tags = []string{} // If JSON unmarshal fails, treat as empty array
		}
	}

	return &gqlmodels.Dao{
		ID:             dbDao.ID,
		ChainID:        int32(dbDao.ChainID),
		ChainName:      dbDao.ChainName,
		Name:           dbDao.Name,
		Code:           dbDao.Code,
		State:          dbDao.State,
		Tags:           tags,
		TimeSyncd:      dbDao.TimeSyncd,
		CountProposals: int32(dbDao.CountProposals),
		Ctime:          dbDao.CTime,
		Utime:          dbDao.UTime,
	}
}

func (s *DaoService) Inspect(baseInput types.BasicInput[string]) (*gqlmodels.Dao, error) {
	code := baseInput.Input
	var dbDao dbmodels.Dao
	err := s.db.Where("code = ?", code).First(&dbDao).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("dao not found")
		}
		return nil, err
	}
	dao := s.convertToGqlDao(dbDao)

	// chips
	chips, err := s.SingleDaoChips(code)
	if err != nil {
		return nil, err
	}
	dao.Chips = chips

	// Query user's liked and subscribed status if user is logged in
	var userID string
	if baseInput.User != nil {
		userID = baseInput.User.Id
	}
	userLikedDaos, userSubscribedDaos, err := s.getUserDaoInteractions(userID, []string{code})
	if err != nil {
		return nil, err
	}

	liked := userLikedDaos[code]
	subscribed := userSubscribedDaos[code]

	dao.Liked = &liked
	dao.Subscribed = &subscribed

	return dao, nil
}

func (s *DaoService) SingleDaoChips(code string) ([]*gqlmodels.DaoChip, error) {
	return s.MultipleDaoChips([]string{code})
}

func (s *DaoService) MultipleDaoChips(codes []string) ([]*gqlmodels.DaoChip, error) {
	var dbChips []dbmodels.DgvDaoChip
	err := s.db.Where("dao_code IN ?", codes).Find(&dbChips).Error
	if err != nil {
		return nil, err
	}

	var chips []*gqlmodels.DaoChip
	for _, dbChip := range dbChips {
		chip := &gqlmodels.DaoChip{
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

	return chips, nil
}

func (s *DaoService) getUserDaoInteractions(userID string, daoCodes []string) (map[string]bool, map[string]bool, error) {
	likedService := NewUserLikedDaoService()
	subscribedService := NewUserSubscribedDaoService()

	userLikedDaos, err := likedService.GetUserLikedDaos(userID, daoCodes)
	if err != nil {
		return nil, nil, err
	}

	userSubscribedDaos, err := subscribedService.GetUserSubscribedDaos(userID, daoCodes)
	if err != nil {
		return nil, nil, err
	}

	return userLikedDaos, userSubscribedDaos, nil
}

func (s *DaoService) GetDaos(baseInput types.BasicInput[*string]) ([]*gqlmodels.Dao, error) {
	var dbDaos []dbmodels.Dao
	if err := s.db.Table("dgv_dao").Where("state = ?", "ACTIVE").Order("seq asc").Find(&dbDaos).Error; err != nil {
		return nil, err
	}

	// Extract dao codes from dbDaos
	var daoCodes []string
	for _, dao := range dbDaos {
		daoCodes = append(daoCodes, dao.Code)
	}

	// Batch query chips for all DAOs
	daosChips, err := s.MultipleDaoChips(daoCodes)
	if err != nil {
		return nil, err
	}

	// Batch query user's liked and subscribed DAOs if user is logged in
	var userID string
	if baseInput.User != nil {
		userID = baseInput.User.Id
	}
	userLikedDaos, userSubscribedDaos, err := s.getUserDaoInteractions(userID, daoCodes)
	if err != nil {
		return nil, err
	}

	var daos []*gqlmodels.Dao
	for _, dbDao := range dbDaos {
		// Check if current user liked this DAO
		liked := userLikedDaos[dbDao.Code]
		subscribed := userSubscribedDaos[dbDao.Code]

		// Convert tags from JSON string to string array
		var tags []string
		if dbDao.Tags != "" {
			if err := json.Unmarshal([]byte(dbDao.Tags), &tags); err != nil {
				// If JSON unmarshal fails, treat as empty array
				tags = []string{}
			}
		}

		// Find chips for this DAO
		var chips []*gqlmodels.DaoChip
		for _, daoChip := range daosChips {
			if daoChip.DaoCode == dbDao.Code {
				chips = append(chips, daoChip)
			}
		}

		dao := s.convertToGqlDao(dbDao)

		dao.Liked = &liked
		dao.Subscribed = &subscribed
		dao.Chips = chips

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
			ID:             internal.NextIDString(),
			ChainID:        input.Config.Chain.ID,
			ChainName:      input.Config.Chain.Name,
			Name:           input.Config.Name,
			Code:           input.Code,
			State:          "ACTIVE",
			Tags:           tagsJson,
			ConfigLink:     input.ConfigLink,
			TimeSyncd:      utils.TimePtrNow(),
			CountProposals: input.CountProposals,
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
		existingDao.TimeSyncd = utils.TimePtrNow()
		existingDao.CountProposals = input.CountProposals
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

type UserLikedDaoService struct {
	db *gorm.DB
}

func NewUserLikedDaoService() *UserLikedDaoService {
	return &UserLikedDaoService{
		db: database.GetDB(),
	}
}

// GetUserLikedDaos 获取用户喜欢的 DAO 列表
func (s *UserLikedDaoService) GetUserLikedDaos(userID string, daoCodes []string) (map[string]bool, error) {
	userLikedDaos := make(map[string]bool)

	if userID == "" {
		return userLikedDaos, nil
	}

	var likedRecords []dbmodels.UserLikedDao
	if err := s.db.Where("user_id = ? AND dao_code IN ?", userID, daoCodes).Find(&likedRecords).Error; err != nil {
		return nil, err
	}

	for _, record := range likedRecords {
		userLikedDaos[record.DaoCode] = true
	}

	return userLikedDaos, nil
}

type UserSubscribedDaoService struct {
	db *gorm.DB
}

func NewUserSubscribedDaoService() *UserSubscribedDaoService {
	return &UserSubscribedDaoService{
		db: database.GetDB(),
	}
}

// GetUserSubscribedDaos 获取用户订阅的 DAO 列表
func (s *UserSubscribedDaoService) GetUserSubscribedDaos(userID string, daoCodes []string) (map[string]bool, error) {
	userSubscribedDaos := make(map[string]bool)

	if userID == "" {
		return userSubscribedDaos, nil
	}

	var subscribedRecords []dbmodels.UserSubscribedDao
	if err := s.db.Where("user_id = ? AND dao_code IN ? AND state = ?", userID, daoCodes, "SUBSCRIBED").Find(&subscribedRecords).Error; err != nil {
		return nil, err
	}

	for _, record := range subscribedRecords {
		userSubscribedDaos[record.DaoCode] = true
	}

	return userSubscribedDaos, nil
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
