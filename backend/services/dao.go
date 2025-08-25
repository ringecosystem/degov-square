package services

import (
	"encoding/json"
	"errors"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/jinzhu/copier"
	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
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

func (s *DaoService) filterAndConvertToGqlProposal(proposals []dbmodels.ProposalTracking, daoCode string) *gqlmodels.Proposal {
	for _, proposal := range proposals {
		if proposal.DaoCode == daoCode {
			gqlProposal := gqlmodels.Proposal{}
			copier.Copy(&gqlProposal, &proposal)
			return &gqlProposal
		}
	}
	return nil
}

func (s *DaoService) convertToGqlDao(dbDao dbmodels.Dao) *gqlmodels.Dao {
	var tags []string
	if dbDao.Tags != "" {
		if err := json.Unmarshal([]byte(dbDao.Tags), &tags); err != nil {
			tags = []string{} // If JSON unmarshal fails, treat as empty array
		}
	}
	gqlDao := gqlmodels.Dao{}
	copier.Copy(&gqlDao, &dbDao)
	gqlDao.Tags = tags
	return &gqlDao
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
	// var userID string
	// if baseInput.User != nil {
	// 	userID = baseInput.User.Id
	// }
	// userLikedDaos, userSubscribedDaos, err := s.getUserDaoInteractions(userID, []string{code})
	// if err != nil {
	// 	return nil, err
	// }

	// liked := userLikedDaos[code]
	// subscribed := userSubscribedDaos[code]

	// dao.Liked = &liked
	// dao.Subscribed = &subscribed

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
		chip := gqlmodels.DaoChip{}
		copier.Copy(&chip, &dbChip)
		chips = append(chips, &chip)
	}

	return chips, nil
}

// func (s *DaoService) getUserDaoInteractions(userID string, daoCodes []string) (map[string]bool, map[string]bool, error) {
// 	likedService := NewUserLikedDaoService()
// 	subscribedService := NewUserSubscribedDaoService()

// 	userLikedDaos, err := likedService.GetUserLikedDaos(userID, daoCodes)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	userSubscribedDaos, err := subscribedService.GetUserSubscribedDaos(userID, daoCodes)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	return userLikedDaos, userSubscribedDaos, nil
// }

func (s *DaoService) ListDaos(baseInput types.BasicInput[*types.ListDaosInput]) ([]*gqlmodels.Dao, error) {
	var dbDaos []dbmodels.Dao

	query := s.db.Table("dgv_dao")

	// If codes are provided, filter by them
	if baseInput.Input != nil && baseInput.Input.Codes != nil && len(*baseInput.Input.Codes) > 0 {
		query = query.Where("code IN ?", *baseInput.Input.Codes)
	}
	if baseInput.Input != nil && baseInput.Input.State != nil && len(*baseInput.Input.State) > 0 {
		query = query.Where("state IN ?", *baseInput.Input.State)
	} else {
		query = query.Where("state = ?", dbmodels.DaoStateActive)
	}

	if err := query.Order("seq asc").Find(&dbDaos).Error; err != nil {
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

	proposalResults, err := s.lastProposalMultiDaos(daoCodes)
	if err != nil {
		return nil, err
	}

	// Batch query user's liked and subscribed DAOs if user is logged in
	// var userID string
	// if baseInput.User != nil {
	// 	userID = baseInput.User.Id
	// }
	// userLikedDaos, userSubscribedDaos, err := s.getUserDaoInteractions(userID, daoCodes)
	// if err != nil {
	// 	return nil, err
	// }

	var daos []*gqlmodels.Dao
	for _, dbDao := range dbDaos {
		// Check if current user liked this DAO
		// liked := userLikedDaos[dbDao.Code]
		// subscribed := userSubscribedDaos[dbDao.Code]

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

		// dao.Liked = &liked
		// dao.Subscribed = &subscribed
		dao.Chips = chips

		dao.LastProposal = s.filterAndConvertToGqlProposal(proposalResults, dbDao.Code)

		daos = append(daos, dao)
	}

	return daos, nil
}

func (s *DaoService) lastProposalMultiDaos(daoCodes []string) ([]dbmodels.ProposalTracking, error) {
	if len(daoCodes) == 0 {
		return nil, nil
	}

	var results []dbmodels.ProposalTracking
	err := s.db.Raw(`
			WITH RankedProposals AS (
					SELECT
							*,
							ROW_NUMBER() OVER(PARTITION BY dao_code ORDER BY proposal_created_at DESC) as rn
					FROM
							dgv_proposal_tracking
			)
			SELECT
				id,
				dao_code,
				chain_id,
				proposal_link,
				proposal_id,
				state,
				proposal_at_block,
				proposal_created_at,
				times_track,
				time_next_track,
				ctime,
				utime
			FROM
					RankedProposals
			WHERE
					rn = 1 AND dao_code IN ?
    `, daoCodes).Scan(&results).Error

	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *DaoService) RefreshDaoAndConfig(input types.RefreshDaoAndConfigInput) error {
	var existingDao dbmodels.Dao
	result := s.db.Where("code = ?", input.Code).First(&existingDao)

	tagsJson := utils.ToJSON(input.Tags)
	if result.Error == gorm.ErrRecordNotFound {
		// Insert new DAO
		dao := &dbmodels.Dao{
			ID:                utils.NextIDString(),
			ChainID:           input.Config.Chain.ID,
			ChainName:         input.Config.Chain.Name,
			ChainLogo:         input.Config.Chain.Logo,
			Name:              input.Config.Name,
			Code:              input.Code,
			Logo:              input.Config.Logo,
			Endpoint:          input.Config.SiteURL,
			State:             input.State,
			Tags:              tagsJson,
			ConfigLink:        input.ConfigLink,
			TimeSyncd:         utils.TimePtrNow(),
			LastTrackingBlock: 0, // Default to 0 for new DAOs
		}

		// Set metrics fields if they are provided (not nil)
		if input.MetricsCountProposals != nil {
			dao.MetricsCountProposals = *input.MetricsCountProposals
		}
		if input.MetricsCountMembers != nil {
			dao.MetricsCountMembers = *input.MetricsCountMembers
		}
		if input.MetricsSumPower != nil {
			dao.MetricsSumPower = *input.MetricsSumPower
		}
		if input.MetricsCountVote != nil {
			dao.MetricsCountVote = *input.MetricsCountVote
		}
		if err := s.db.Create(dao).Error; err != nil {
			return err
		}
	} else {
		// Update existing DAO
		existingDao.ChainID = input.Config.Chain.ID
		existingDao.ChainName = input.Config.Chain.Name
		existingDao.ChainLogo = input.Config.Chain.Logo
		existingDao.Name = input.Config.Name
		existingDao.Logo = input.Config.Logo
		existingDao.Endpoint = input.Config.SiteURL
		existingDao.State = input.State
		existingDao.Tags = tagsJson
		existingDao.ConfigLink = input.ConfigLink
		existingDao.UTime = utils.TimePtrNow()
		existingDao.TimeSyncd = utils.TimePtrNow()

		// Only update metrics if they are provided (not nil)
		if input.MetricsCountProposals != nil {
			existingDao.MetricsCountProposals = *input.MetricsCountProposals
		}
		if input.MetricsCountMembers != nil {
			existingDao.MetricsCountMembers = *input.MetricsCountMembers
		}
		if input.MetricsSumPower != nil {
			existingDao.MetricsSumPower = *input.MetricsSumPower
		}
		if input.MetricsCountVote != nil {
			existingDao.MetricsCountVote = *input.MetricsCountVote
		}
		if err := s.db.Save(&existingDao).Error; err != nil {
			return err
		}
	}

	var existingConfig dbmodels.DgvDaoConfig
	r2 := s.db.Where("dao_code = ?", input.Code).First(&existingConfig)

	if r2.Error == gorm.ErrRecordNotFound {
		// Insert new DAO config
		config := &dbmodels.DgvDaoConfig{
			ID:      utils.NextIDString(),
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
		Where("code NOT IN ? AND state != ?", getMapKeys(activeCodes), dbmodels.DaoStateInactive).
		Updates(map[string]interface{}{
			"state": dbmodels.DaoStateInactive,
			"utime": utils.TimePtrNow(),
		})

	if result.Error != nil {
		return result.Error
	}

	return nil
}

// UpdateDaoLastTrackingBlock updates the last tracking block for a DAO
func (s *DaoService) UpdateDaoLastTrackingBlock(daoCode string, blockNumber int) error {
	return s.db.Model(&dbmodels.Dao{}).
		Where("code = ?", daoCode).
		Update("last_tracking_block", blockNumber).Error
}

// getMapKeys extracts keys from a map[string]bool
func getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

type DaoConfigService struct {
	db *gorm.DB
}

func NewDaoConfigService() *DaoConfigService {
	return &DaoConfigService{
		db: database.GetDB(),
	}
}

func (s *DaoConfigService) Inspect(daoCode string) (*dbmodels.DgvDaoConfig, error) {
	var config dbmodels.DgvDaoConfig
	err := s.db.Where("dao_code = ?", daoCode).First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("dao config not found")
		}
		return nil, err
	}
	return &config, nil
}

func (s *DaoConfigService) RawConfig(input gqlmodels.GetDaoConfigInput) (string, error) {
	daoConfig, err := s.Inspect(input.DaoCode)
	if err != nil {
		return "", err
	}

	format := gqlmodels.ConfigFormatYaml
	if input.Format != nil {
		format = *input.Format
	}

	if format == gqlmodels.ConfigFormatJSON {
		// Convert YAML to JSON
		var yamlData interface{}
		err := yaml.Unmarshal([]byte(daoConfig.Config), &yamlData)
		if err != nil {
			return "", errors.New("failed to convert YAML to JSON")
		}

		jsonData, err := json.MarshalIndent(yamlData, "", "  ")
		if err != nil {
			return "", errors.New("failed to convert YAML to JSON")
		}

		return string(jsonData), nil
	} else {
		// Default to YAML format
		return daoConfig.Config, nil
	}
}

type UserLikedDaoService struct {
	db         *gorm.DB
	daoService *DaoService
}

func NewUserLikedDaoService() *UserLikedDaoService {
	return &UserLikedDaoService{
		db:         database.GetDB(),
		daoService: NewDaoService(),
	}
}

// func (s *DaoService) LikedDaos(baseInput types.BasicInput[*string]) ([]*gqlmodels.Dao, error) {
// 	// Get user ID from input
// 	userID := ""
// 	if baseInput.User != nil {
// 		userID = baseInput.User.Id
// 	}

// 	// Get liked DAO codes for the user
// 	likedDaos, err := s.likedService.GetUserLikedDaos(userID, nil) // Pass nil to get all liked DAOs
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Convert liked DAO codes to a slice
// 	var daoCodes []string
// 	for code := range likedDaos {
// 		daoCodes = append(daoCodes, code)
// 	}

// 	// Fetch DAOs by codes
// 	return s.ListDaos(types.BasicInput[*types.ListDaosInput]{Input: &types.ListDaosInput{
// 		Codes: &daoCodes,
// 	}})
// }

// liked daos
func (s *UserLikedDaoService) LikedDaos(baseInput types.BasicInput[*string]) ([]*gqlmodels.Dao, error) {
	if baseInput.User == nil {
		return []*gqlmodels.Dao{}, nil
	}

	// First get liked dao codes by joining with dgv_dao to ensure only ACTIVE DAOs
	var daoCodes []string
	err := s.db.Table("dgv_user_liked_dao").
		Select("dgv_user_liked_dao.dao_code").
		Joins("INNER JOIN dgv_dao ON dgv_user_liked_dao.dao_code = dgv_dao.code").
		Where("dgv_user_liked_dao.user_id = ? AND dgv_dao.state = ?", baseInput.User.Id, "ACTIVE").
		Pluck("dao_code", &daoCodes).Error

	if err != nil {
		return nil, err
	}

	// If no liked DAOs found, return empty array
	if len(daoCodes) == 0 {
		return []*gqlmodels.Dao{}, nil
	}

	// Use existing ListDaos method to get the DAOs with all their data
	return s.daoService.ListDaos(types.BasicInput[*types.ListDaosInput]{
		Input: &types.ListDaosInput{
			Codes: &daoCodes,
		},
		User: baseInput.User,
	})
}

type UserSubscribedDaoService struct {
	db         *gorm.DB
	daoService *DaoService
}

func NewUserSubscribedDaoService() *UserSubscribedDaoService {
	return &UserSubscribedDaoService{
		db:         database.GetDB(),
		daoService: NewDaoService(),
	}
}

func (s *UserSubscribedDaoService) SubscribedDaos(baseInput types.BasicInput[*string]) ([]*gqlmodels.Dao, error) {
	if baseInput.User == nil {
		return []*gqlmodels.Dao{}, nil
	}

	// First get subscribed dao codes by joining with dgv_dao to ensure only ACTIVE DAOs
	var daoCodes []string
	err := s.db.Table("dgv_user_subscribed_dao").
		Select("dgv_user_subscribed_dao.dao_code").
		Joins("INNER JOIN dgv_dao ON dgv_user_subscribed_dao.dao_code = dgv_dao.code").
		Where("dgv_user_subscribed_dao.user_id = ? AND dgv_user_subscribed_dao.state = ? AND dgv_dao.state = ?",
			baseInput.User.Id, "SUBSCRIBED", "ACTIVE").
		Pluck("dao_code", &daoCodes).Error

	if err != nil {
		return nil, err
	}

	// If no subscribed DAOs found, return empty array
	if len(daoCodes) == 0 {
		return []*gqlmodels.Dao{}, nil
	}

	// Use existing ListDaos method to get the DAOs with all their data
	return s.daoService.ListDaos(types.BasicInput[*types.ListDaosInput]{
		Input: &types.ListDaosInput{
			Codes: &daoCodes,
		},
		User: baseInput.User,
	})
}

// func (s *UserSubscribedDaoService) GetUserSubscribedDaos(userID string, daoCodes []string) (map[string]bool, error) {
// 	userSubscribedDaos := make(map[string]bool)

// 	if userID == "" {
// 		return userSubscribedDaos, nil
// 	}

// 	var subscribedRecords []dbmodels.UserSubscribedDao
// 	if err := s.db.Where("user_id = ? AND dao_code IN ? AND state = ?", userID, daoCodes, "SUBSCRIBED").Find(&subscribedRecords).Error; err != nil {
// 		return nil, err
// 	}

// 	for _, record := range subscribedRecords {
// 		userSubscribedDaos[record.DaoCode] = true
// 	}

// 	return userSubscribedDaos, nil
// }

type DaoChipService struct {
	db *gorm.DB
}

func NewDaoChipService() *DaoChipService {
	return &DaoChipService{
		db: database.GetDB(),
	}
}

func (s *DaoChipService) SyncAgentChips(agentDaoConfigs []types.AgentDaoConfig) error {
	chipCode := dbmodels.ChipCodeAgent

	// Build map for quick lookup and prepare new chips
	agentDaoMap := make(map[string]types.AgentDaoConfig, len(agentDaoConfigs))
	newChips := make([]dbmodels.DgvDaoChip, 0, len(agentDaoConfigs))

	for _, agentDao := range agentDaoConfigs {
		agentDaoMap[agentDao.Code] = agentDao
		newChips = append(newChips, dbmodels.DgvDaoChip{
			ID:         utils.NextIDString(),
			DaoCode:    agentDao.Code,
			ChipCode:   chipCode,
			Flag:       "ENABLED",
			Additional: utils.ToJSON(agentDao),
			CTime:      time.Now(),
		})
	}

	// Use transaction for atomicity
	return s.db.Transaction(func(tx *gorm.DB) error {
		// First, delete all existing agent chips
		if err := tx.Where("chip_code = ?", chipCode).Delete(&dbmodels.DgvDaoChip{}).Error; err != nil {
			return err
		}

		// Then, batch insert new chips if any
		if len(newChips) > 0 {
			if err := tx.Create(&newChips).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *DaoChipService) StoreChipMetricsState(input types.StoreDaoChipMetricsStateInput) error {
	chipCode := dbmodels.ChipCodeMetricsState

	// Extract unique dao_codes from input.MetricsStates
	daoCodesMap := make(map[string]struct{})
	for _, state := range input.MetricsStates {
		daoCodesMap[state.DaoCode] = struct{}{}
	}

	// Convert to slice
	daoCodes := make([]string, 0, len(daoCodesMap))
	for daoCode := range daoCodesMap {
		daoCodes = append(daoCodes, daoCode)
	}

	// Delete existing records for these dao_codes with chip_code = METRICS_STATE
	if len(daoCodes) > 0 {
		if err := s.db.Where("dao_code IN ? AND chip_code = ?", daoCodes, chipCode).Delete(&dbmodels.DgvDaoChip{}).Error; err != nil {
			return err
		}
	}

	// Batch insert new records
	chips := make([]dbmodels.DgvDaoChip, 0, len(input.MetricsStates))
	for _, state := range input.MetricsStates {
		chip := dbmodels.DgvDaoChip{
			ID:         utils.NextIDString(),
			DaoCode:    state.DaoCode,
			ChipCode:   chipCode,
			Flag:       string(state.State), // MetricsState.State as flag
			Additional: utils.ToJSON(state), // Single MetricsState as JSON
			CTime:      time.Now(),
			UTime:      utils.TimePtrNow(),
		}
		chips = append(chips, chip)
	}

	// Batch insert
	if len(chips) > 0 {
		if err := s.db.Create(&chips).Error; err != nil {
			return err
		}
	}

	return nil
}
