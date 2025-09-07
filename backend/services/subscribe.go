package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/internal/utils"
	"github.com/ringecosystem/degov-apps/types"
	"gorm.io/gorm"
)

type SubscribeService struct {
	db         *gorm.DB
	daoService *DaoService
}

func NewSubscribeService() *SubscribeService {
	return &SubscribeService{
		db:         database.GetDB(),
		daoService: NewDaoService(),
	}
}

type buildFeatureInput struct {
	ChainID         int
	DaoCode         string
	ProposalID      *string
	UserID          string
	UserAddress     string
	FeatureSettings []*gqlmodels.FeatureSettingsInput
}

type resetDaoFeaturesInput struct {
	DaoCode  string
	UserID   string
	Features []dbmodels.SubscribeFeature
}

// type buildProposalFeatureInput struct {
// 	ChainID         int
// 	DaoCode         string
// 	ProposalID      string
// 	UserID          string
// 	UserAddress     string
// 	FeatureSettings []*gqlmodels.FeatureSettingsInput
// }

type resetProposalFeaturesInput struct {
	DaoCode    string
	UserID     string
	ProposalID string
	Features   []dbmodels.SubscribeFeature
}

type InspectSubscribeProposalInput struct {
	DaoCode    string
	ProposalID string
}

func (s *SubscribeService) buildFeatures(
	input *buildFeatureInput,
) []dbmodels.SubscribeFeature {
	featureSettings := input.FeatureSettings
	var features []dbmodels.SubscribeFeature

	if featureSettings == nil {
		return features
	}

	for _, featureSetting := range featureSettings {
		if featureSetting == nil {
			continue
		}

		var dbFeatureName dbmodels.SubscribeFeatureName
		var strategy string

		switch featureSetting.Name {
		case gqlmodels.FeatureNameVoteEnd:
			dbFeatureName = dbmodels.SubscribeFeatureVoteEnd
		case gqlmodels.FeatureNameVoteEmitted:
			dbFeatureName = dbmodels.SubscribeFeatureVoteEmitted
		case gqlmodels.FeatureNameProposalStateChanged:
			dbFeatureName = dbmodels.SubscribeFeatureProposalStateChanged
		case gqlmodels.FeatureNameProposalNew:
			dbFeatureName = dbmodels.SubscribeFeatureProposalNew
			continue
		default:
			// skip unsupported feature
			continue
		}

		if featureSetting.Strategy != nil && *featureSetting.Strategy != "" {
			strategy = *featureSetting.Strategy
		} else {
			strategy = "true"
		}

		features = append(features, dbmodels.SubscribeFeature{
			ID:          utils.NextIDString(),
			ChainID:     input.ChainID,
			DaoCode:     input.DaoCode,
			ProposalID:  input.ProposalID,
			UserID:      input.UserID,
			UserAddress: input.UserAddress,
			Feature:     dbFeatureName,
			Strategy:    strategy,
		})
	}

	return features
}

func (s *SubscribeService) SubscribeDao(baseInput types.BasicInput[gqlmodels.SubscribeDaoInput]) (*gqlmodels.SubscribedDaoOutput, error) {
	user := baseInput.User
	sdInput := baseInput.Input
	featureSettings := sdInput.Features

	existingDao, err := s.daoService.Inspect(types.BasicInput[string]{
		User:  user,
		Input: sdInput.DaoCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect existing DAO: %w", err)
	}

	existingSubscribedDao, err1 := s.InspectSubscribeDao(types.BasicInput[string]{
		User:  user,
		Input: sdInput.DaoCode,
	})
	chainId := int(existingDao.ChainID)

	if err1 == nil {
		existingSubscribedDao.UTime = time.Now()
		if err := s.db.Save(existingSubscribedDao).Error; err != nil {
			return nil, err
		}
	} else {
		newSubscribedDao := dbmodels.UserSubscribedDao{
			ID:          utils.NextIDString(),
			ChainID:     chainId,
			DaoCode:     sdInput.DaoCode,
			UserID:      user.Id,
			UserAddress: user.Address,
			State:       dbmodels.SubscribeStateActive,
		}
		if err := s.db.Create(&newSubscribedDao).Error; err != nil {
			return nil, err
		}
	}

	features := s.buildFeatures(&buildFeatureInput{
		ChainID:         chainId,
		DaoCode:         sdInput.DaoCode,
		UserID:          user.Id,
		UserAddress:     user.Address,
		FeatureSettings: featureSettings,
	})

	if err := s.resetDaoFeatures(resetDaoFeaturesInput{
		UserID:   user.Id,
		DaoCode:  sdInput.DaoCode,
		Features: features,
	}); err != nil {
		return nil, err
	}

	output := &gqlmodels.SubscribedDaoOutput{
		DaoCode: sdInput.DaoCode,
		State:   string(dbmodels.SubscribeStateActive),
	}
	return output, nil
}

func (s *SubscribeService) UnsubscribeDao(baseInput types.BasicInput[gqlmodels.UnsubscribeDaoInput]) (*gqlmodels.SubscribedDaoOutput, error) {
	user := baseInput.User
	input := baseInput.Input

	existingSubscribedDao, err := s.InspectSubscribeDao(types.BasicInput[string]{
		User:  user,
		Input: input.DaoCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect existing subscribed DAO: %w", err)
	}

	if err := s.db.Update("state", dbmodels.SubscribeStateInactive).Where("id = ?", existingSubscribedDao.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to unsusbscribe dao %w", err)
	}

	output := &gqlmodels.SubscribedDaoOutput{
		DaoCode: input.DaoCode,
		State:   string(dbmodels.SubscribeStateInactive),
	}
	return output, nil
}

func (s *SubscribeService) InspectSubscribeDao(baseInput types.BasicInput[string]) (*dbmodels.UserSubscribedDao, error) {
	user := baseInput.User
	daoCode := baseInput.Input

	var subscribedDao dbmodels.UserSubscribedDao
	err := s.db.
		Where(
			"(user_id = ? or user_address=?) AND dao_code = ?",
			user.Id,
			user.Address,
			daoCode,
		).
		First(&subscribedDao).Error
	if err != nil {
		return nil, err
	}
	return &subscribedDao, nil
}

func (s *SubscribeService) InspectSubscribeProposal(baseInput types.BasicInput[InspectSubscribeProposalInput]) (*dbmodels.UserSubscribedProposal, error) {
	user := baseInput.User
	input := baseInput.Input

	var subscribedProposal dbmodels.UserSubscribedProposal
	err := s.db.
		Where(
			"(user_id = ? or user_address= ?) AND dao_code = ? AND proposal_id = ?",
			user.Id,
			user.Address,
			input.DaoCode,
			input.ProposalID,
		).
		First(&subscribedProposal).Error
	if err != nil {
		return nil, err
	}
	return &subscribedProposal, nil
}

func (s *SubscribeService) SubscribeProposal(baseInput types.BasicInput[gqlmodels.SubscribeProposalInput]) (*gqlmodels.SubscribedProposalOutput, error) {
	user := baseInput.User
	spInput := baseInput.Input
	featureSettings := spInput.Features

	existingDao, err := s.daoService.Inspect(types.BasicInput[string]{
		User:  user,
		Input: spInput.DaoCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect existing DAO: %w", err)
	}
	chainId := int(existingDao.ChainID)

	// check existing subscribed proposal
	existingSubscribedProposal, err1 := s.InspectSubscribeProposal(types.BasicInput[InspectSubscribeProposalInput]{
		User: user,
		Input: InspectSubscribeProposalInput{
			DaoCode:    spInput.DaoCode,
			ProposalID: spInput.ProposalID,
		},
	})
	if err1 == nil {
		existingSubscribedProposal.UTime = time.Now()
		if err := s.db.Save(existingSubscribedProposal).Error; err != nil {
			return nil, err
		}
	} else {
		newSubscribedProposal := dbmodels.UserSubscribedProposal{
			ID:          utils.NextIDString(),
			ChainID:     chainId,
			DaoCode:     spInput.DaoCode,
			UserID:      user.Id,
			UserAddress: user.Address,
			State:       dbmodels.SubscribeStateActive,
			ProposalID:  spInput.ProposalID,
		}
		if err := s.db.Create(&newSubscribedProposal).Error; err != nil {
			return nil, err
		}
	}

	features := s.buildFeatures(&buildFeatureInput{
		ChainID:         chainId,
		DaoCode:         spInput.DaoCode,
		ProposalID:      &spInput.ProposalID,
		UserID:          user.Id,
		UserAddress:     user.Address,
		FeatureSettings: featureSettings,
	})

	if err := s.resetProposalFeatures(resetProposalFeaturesInput{
		UserID:     user.Id,
		DaoCode:    spInput.DaoCode,
		ProposalID: spInput.ProposalID,
		Features:   features,
	}); err != nil {
		return nil, err
	}

	output := &gqlmodels.SubscribedProposalOutput{
		DaoCode:    spInput.DaoCode,
		ProposalID: spInput.ProposalID,
		State:      string(dbmodels.SubscribeStateActive),
	}
	return output, nil
}

func (s *SubscribeService) UnsubscribeProposal(baseInput types.BasicInput[gqlmodels.UnsubscribeProposalInput]) (*gqlmodels.SubscribedProposalOutput, error) {
	user := baseInput.User
	input := baseInput.Input

	existingSubscribedProposal, err := s.InspectSubscribeProposal(types.BasicInput[InspectSubscribeProposalInput]{
		User: user,
		Input: InspectSubscribeProposalInput{
			DaoCode:    input.DaoCode,
			ProposalID: input.ProposalID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect existing subscribed proposal: %w", err)
	}

	if err := s.db.Update("state", dbmodels.SubscribeStateInactive).Where("id = ?", existingSubscribedProposal.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to unsubscribe proposal: %w", err)
	}

	output := &gqlmodels.SubscribedProposalOutput{
		DaoCode:    input.DaoCode,
		ProposalID: input.ProposalID,
		State:      string(dbmodels.SubscribeStateInactive),
	}
	return output, nil
}

func (s *SubscribeService) ListSubscribedUser(input types.ListSubscribeUserInput) ([]types.ListSubscribedUserOutput, error) {
	strategies := input.Strategies
	if len(strategies) == 0 {
		return nil, fmt.Errorf("no strategies provided for feature %s", input.Feature)
	}

	queryParams := make([]interface{}, 0)
	whereConditions := make([]string, 0)

	whereConditions = append(whereConditions, "f.feature = ?", "f.strategy IN ?", "f.dao_code = ?")
	queryParams = append(queryParams, input.Feature, strategies, input.DaoCode)

	if input.ProposalID != nil {
		whereConditions = append(whereConditions, "(f.proposal_id = ? OR f.proposal_id IS NULL)")
		queryParams = append(queryParams, *input.ProposalID)
	} else {
		whereConditions = append(whereConditions, "f.proposal_id IS NULL")
	}

	if input.TimeEvent != nil {
		whereConditions = append(whereConditions, "((d.state = 'ACTIVE' AND d.ctime <= ?) OR (p.state = 'ACTIVE' AND p.ctime <= ?))")
		queryParams = append(queryParams, *input.TimeEvent, *input.TimeEvent)
	} else {
		whereConditions = append(whereConditions, "(d.state = 'ACTIVE' OR p.state = 'ACTIVE')")
	}

	sqlTemplate := `
WITH RankedResults AS (
    SELECT
        f.user_id, f.user_address, f.chain_id, f.dao_code,
        LEAST(d.ctime, p.ctime) AS ctime,
        f.ctime AS order_ctime,
        ROW_NUMBER() OVER(
            PARTITION BY f.user_id, f.user_address, f.chain_id, f.dao_code, LEAST(d.ctime, p.ctime)
            ORDER BY f.ctime ASC, f.user_id ASC
        ) as rn
    FROM
        dgv_subscribed_feature AS f
    LEFT JOIN
        dgv_user_subscribed_dao AS d ON f.user_id = d.user_id AND f.dao_code = d.dao_code
    LEFT JOIN
        dgv_user_subscribed_proposal AS p ON f.user_id = p.user_id AND f.proposal_id = p.proposal_id
    WHERE
        %s
)
SELECT user_id, user_address, chain_id, dao_code, ctime
FROM RankedResults
WHERE rn = 1
ORDER BY order_ctime ASC, user_id ASC
LIMIT ? OFFSET ?
`

	whereClause := strings.Join(whereConditions, " AND ")
	finalSQL := fmt.Sprintf(sqlTemplate, whereClause)

	if input.Limit <= 0 {
		input.Limit = 100
	}
	queryParams = append(queryParams, input.Limit, input.Offset)

	var outputs []types.ListSubscribedUserOutput
	if err := s.db.Raw(finalSQL, queryParams...).Scan(&outputs).Error; err != nil {
		return nil, err
	}

	return outputs, nil
}

func (s *SubscribeService) resetDaoFeatures(input resetDaoFeaturesInput) error {
	if err := s.db.Where(
		"dao_code = ? and user_id =?",
		input.DaoCode,
		input.UserID,
	).
		Delete(&dbmodels.SubscribeFeature{}).Error; err != nil {
		return err
	}

	if len(input.Features) > 0 {
		return s.db.Create(input.Features).Error
	}
	return nil
}

// resetProposalFeatures deletes existing features scoped to the dao + proposal + user
// and inserts the provided features (if any).
func (s *SubscribeService) resetProposalFeatures(input resetProposalFeaturesInput) error {
	if err := s.db.Where(
		"dao_code = ? and proposal_id = ? and (user_id = ? or user_address = ?)",
		input.DaoCode,
		input.ProposalID,
		input.UserID,
		input.UserID,
	).
		Delete(&dbmodels.SubscribeFeature{}).Error; err != nil {
		return err
	}

	if len(input.Features) > 0 {
		return s.db.Create(input.Features).Error
	}
	return nil
}

func (s *SubscribeService) ListFeatures(baseInput types.BasicInput[types.ListFeaturesInput]) ([]dbmodels.SubscribeFeature, error) {
	if baseInput.User == nil {
		return nil, fmt.Errorf("not logged in")
	}

	query := s.db.Model(&dbmodels.SubscribeFeature{}).
		Where("user_id = ?", baseInput.User.Id).
		Where("dao_code = ?", baseInput.Input.DaoCode)

	if baseInput.Input.ProposalID != nil && *baseInput.Input.ProposalID != "" {
		query = query.Where("proposal_id = ?", *baseInput.Input.ProposalID)
	} else {
		query = query.Where("proposal_id IS NULL OR proposal_id = ''")
	}

	var features []dbmodels.SubscribeFeature
	if err := query.Find(&features).Error; err != nil {
		return nil, err
	}

	return features, nil
}

type UserSubscribedDaoService struct {
	db               *gorm.DB
	daoService       *DaoService
	subscribeService *SubscribeService
	proposalService  *ProposalService
}

func NewUserSubscribedDaoService() *UserSubscribedDaoService {
	return &UserSubscribedDaoService{
		db:               database.GetDB(),
		daoService:       NewDaoService(),
		subscribeService: NewSubscribeService(),
		proposalService:  NewProposalService(),
	}
}

func (s *UserSubscribedDaoService) SubscribedDaos(baseInput types.BasicInput[*string]) ([]*gqlmodels.SubscribedDao, error) {
	if baseInput.User == nil {
		return nil, fmt.Errorf("not logged in")
	}

	// First get subscribed dao codes by joining with dgv_dao to ensure only ACTIVE DAOs
	var daoCodes []string
	err := s.db.Table("dgv_user_subscribed_dao").
		Select("dgv_user_subscribed_dao.dao_code").
		Joins("INNER JOIN dgv_dao ON dgv_user_subscribed_dao.dao_code = dgv_dao.code").
		Where("dgv_user_subscribed_dao.user_id = ? AND dgv_user_subscribed_dao.state = ? AND dgv_dao.state = ?",
			baseInput.User.Id, "ACTIVE", "ACTIVE").
		Pluck("dao_code", &daoCodes).
		Error
	if err != nil {
		return nil, err
	}

	// If no subscribed DAOs found, return empty array
	if len(daoCodes) == 0 {
		return []*gqlmodels.SubscribedDao{}, nil
	}

	// Use existing ListDaos method to get the DAOs with all their data
	daos, err := s.daoService.ListDaos(types.BasicInput[*types.ListDaosInput]{
		Input: &types.ListDaosInput{
			Codes: &daoCodes,
		},
		User: baseInput.User,
	})
	if err != nil {
		return nil, err
	}

	results := []*gqlmodels.SubscribedDao{}

	for _, dao := range daos {
		features, err := s.subscribeService.ListFeatures(types.BasicInput[types.ListFeaturesInput]{
			User: baseInput.User,
			Input: types.ListFeaturesInput{
				DaoCode:    dao.Code,
				ProposalID: nil,
			},
		})
		if err != nil {
			return nil, err
		}

		subscribeFeatures := []*gqlmodels.SubscribedFeature{}
		for _, f := range features {
			subscribeFeatures = append(subscribeFeatures, &gqlmodels.SubscribedFeature{
				Name:     string(f.Feature),
				Strategy: f.Strategy,
			})
		}

		results = append(results, &gqlmodels.SubscribedDao{
			Dao:      dao,
			Features: subscribeFeatures,
		})
	}

	return results, nil
}

func (s *UserSubscribedDaoService) SubscribedProposals(baseInput types.BasicInput[*string]) ([]*gqlmodels.SubscribedProposal, error) {
	if baseInput.User == nil {
		return nil, fmt.Errorf("not logged in")
	}

	// First get subscribed proposals by joining with dgv_dao to ensure only ACTIVE DAOs
	var subscribedProposalData []struct {
		DaoCode    string
		ProposalID string
	}
	err := s.db.Table("dgv_user_subscribed_proposal").
		Select("dgv_user_subscribed_proposal.dao_code, dgv_user_subscribed_proposal.proposal_id").
		Joins("INNER JOIN dgv_dao ON dgv_user_subscribed_proposal.dao_code = dgv_dao.code").
		Where("dgv_user_subscribed_proposal.user_id = ? AND dgv_user_subscribed_proposal.state = ? AND dgv_dao.state = ?",
			baseInput.User.Id, "ACTIVE", "ACTIVE").
		Scan(&subscribedProposalData).
		Error
	if err != nil {
		return nil, err
	}

	// If no subscribed proposals found, return empty array
	if len(subscribedProposalData) == 0 {
		return []*gqlmodels.SubscribedProposal{}, nil
	}

	results := []*gqlmodels.SubscribedProposal{}

	for _, spd := range subscribedProposalData {
		dao, err := s.daoService.Inspect(types.BasicInput[string]{
			User:  baseInput.User,
			Input: spd.DaoCode,
		})
		if err != nil {
			return nil, err
		}
		proposal, err := s.proposalService.InspectProposal(types.InspectProposalInput{
			DaoCode:    spd.DaoCode,
			ProposalID: spd.ProposalID,
		})
		if err != nil {
			return nil, err
		}
		gqlProposal := s.proposalService.ConvertToGqlProposal(proposal)

		features, err := s.subscribeService.ListFeatures(types.BasicInput[types.ListFeaturesInput]{
			User: baseInput.User,
			Input: types.ListFeaturesInput{
				DaoCode:    spd.DaoCode,
				ProposalID: &spd.ProposalID,
			},
		})
		if err != nil {
			return nil, err
		}

		subscribeFeatures := []*gqlmodels.SubscribedFeature{}
		for _, f := range features {
			subscribeFeatures = append(subscribeFeatures, &gqlmodels.SubscribedFeature{
				Name:     string(f.Feature),
				Strategy: f.Strategy,
			})
		}

		results = append(results, &gqlmodels.SubscribedProposal{
			Dao:      dao,
			Proposal: gqlProposal,
			Features: subscribeFeatures,
		})
	}

	return results, nil
}
