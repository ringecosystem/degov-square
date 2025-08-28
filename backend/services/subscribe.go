package services

import (
	"fmt"
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

type buildDaoFeatureInput struct {
	ChainID      int
	DaoCode      string
	UserID       string
	UserAddress  string
	FeatureInput *gqlmodels.FeatureSettingsDaoInput
}

type resetDaoFeaturesInput struct {
	DaoCode  string
	UserID   string
	Features []dbmodels.SubscribeFeature
}

type buildProposalFeatureInput struct {
	ChainID      int
	DaoCode      string
	ProposalID   string
	UserID       string
	UserAddress  string
	FeatureInput *gqlmodels.FeatureSettingsProposalInput
}

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

func (s *SubscribeService) buildDaoFeatures(
	input *buildDaoFeatureInput,
) []dbmodels.SubscribeFeature {
	featureInput := input.FeatureInput
	var features []dbmodels.SubscribeFeature

	if featureInput == nil {
		return features
	}

	// determine feature strategies; default to false -> "disable" when nil
	enableProposal := false
	enableVotingEndReminder := false
	if featureInput.EnableProposal != nil && *featureInput.EnableProposal {
		enableProposal = true
	}
	if featureInput.EnableVotingEndReminder != nil && *featureInput.EnableVotingEndReminder {
		enableVotingEndReminder = true
	}

	proposalStrategy := "disable"
	if enableProposal {
		proposalStrategy = "enable"
	}
	votingEndStrategy := "disable"
	if enableVotingEndReminder {
		votingEndStrategy = "enable"
	}

	if featureInput.EnableProposal != nil {
		features = append(features, dbmodels.SubscribeFeature{
			ID:       utils.NextIDString(),
			ChainID:  input.ChainID,
			DaoCode:  input.DaoCode,
			UserID:   input.UserID,
			Feature:  dbmodels.SubscribeFeatureEnableProposal,
			Strategy: proposalStrategy,
		})
	}

	if featureInput.EnableVotingEndReminder != nil {
		features = append(features, dbmodels.SubscribeFeature{
			ID:       utils.NextIDString(),
			ChainID:  input.ChainID,
			DaoCode:  input.DaoCode,
			UserID:   input.UserID,
			Feature:  dbmodels.SubscribeFeatureEnableVotingEndReminder,
			Strategy: votingEndStrategy,
		})
	}

	return features
}

func (s *SubscribeService) SubscribeDao(baseInput types.BasicInput[gqlmodels.SubscribeDaoInput]) (*gqlmodels.SubscribedDaoOutput, error) {
	user := baseInput.User
	sdInput := baseInput.Input
	featureInput := sdInput.Feature

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
		// You may want to save the updated time to DB here, e.g.:
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
			State:       "SUBSCRIBED",
		}
		if err := s.db.Create(&newSubscribedDao).Error; err != nil {
			return nil, err
		}
	}

	features := s.buildDaoFeatures(&buildDaoFeatureInput{
		ChainID:      chainId,
		DaoCode:      sdInput.DaoCode,
		UserID:       user.Id,
		UserAddress:  user.Address,
		FeatureInput: featureInput,
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
	featureInput := spInput.Feature

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

	features := s.buildProposalFeatures(&buildProposalFeatureInput{
		ChainID:      chainId,
		DaoCode:      spInput.DaoCode,
		ProposalID:   spInput.ProposalID,
		UserID:       user.Id,
		UserAddress:  user.Address,
		FeatureInput: featureInput,
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
	}
	return output, nil
}

func (s *SubscribeService) resetDaoFeatures(input resetDaoFeaturesInput) error {
	if err := s.db.Where(
		"dao_code = ? and (user_id =? or user_address = ?)",
		input.DaoCode,
		input.UserID,
		input.UserID,
	).
		Delete(&dbmodels.SubscribeFeature{}).Error; err != nil {
		return err
	}

	if len(input.Features) > 0 {
		return s.db.CreateInBatches(input.Features, len(input.Features)).Error
	}
	return nil
}

// buildProposalFeatures builds SubscribeFeature records for a proposal based on the
// provided FeatureSettingsProposalInput. It defaults missing booleans to false and
// only creates feature rows for fields that were explicitly provided (non-nil).
func (s *SubscribeService) buildProposalFeatures(input *buildProposalFeatureInput) []dbmodels.SubscribeFeature {
	featureInput := input.FeatureInput
	var features []dbmodels.SubscribeFeature

	if featureInput == nil {
		return features
	}

	// determine feature strategies; default to false -> "disable" when nil
	enableVotingEndReminder := false
	enableVoted := false
	enableStateChanged := false
	if featureInput.EnableVotingEndReminder != nil && *featureInput.EnableVotingEndReminder {
		enableVotingEndReminder = true
	}
	if featureInput.EnableVoted != nil && *featureInput.EnableVoted {
		enableVoted = true
	}
	if featureInput.EnableStateChanged != nil && *featureInput.EnableStateChanged {
		enableStateChanged = true
	}

	votingEndStrategy := "disable"
	if enableVotingEndReminder {
		votingEndStrategy = "enable"
	}
	votedStrategy := "disable"
	if enableVoted {
		votedStrategy = "enable"
	}
	stateChangedStrategy := "disable"
	if enableStateChanged {
		stateChangedStrategy = "enable"
	}

	pid := input.ProposalID
	if featureInput.EnableVotingEndReminder != nil {
		features = append(features, dbmodels.SubscribeFeature{
			ID:          utils.NextIDString(),
			ChainID:     input.ChainID,
			DaoCode:     input.DaoCode,
			UserID:      input.UserID,
			UserAddress: input.UserAddress,
			Feature:     dbmodels.SubscribeFeatureEnableVotingEndReminder,
			Strategy:    votingEndStrategy,
			ProposalID:  &pid,
		})
	}

	if featureInput.EnableVoted != nil {
		features = append(features, dbmodels.SubscribeFeature{
			ID:          utils.NextIDString(),
			ChainID:     input.ChainID,
			DaoCode:     input.DaoCode,
			UserID:      input.UserID,
			UserAddress: input.UserAddress,
			Feature:     dbmodels.SubscribeFeatureEnableVoted,
			Strategy:    votedStrategy,
			ProposalID:  &pid,
		})
	}

	if featureInput.EnableStateChanged != nil {
		features = append(features, dbmodels.SubscribeFeature{
			ID:          utils.NextIDString(),
			ChainID:     input.ChainID,
			DaoCode:     input.DaoCode,
			UserID:      input.UserID,
			UserAddress: input.UserAddress,
			Feature:     dbmodels.SubscribeFeatureEnableStateChanged,
			Strategy:    stateChangedStrategy,
			ProposalID:  &pid,
		})
	}

	return features
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
		return s.db.CreateInBatches(input.Features, len(input.Features)).Error
	}
	return nil
}
