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
	if featureInput != nil {
		if featureInput.EnableProposal != nil && *featureInput.EnableProposal {
			enableProposal = true
		}
		if featureInput.EnableVotingEndReminder != nil && *featureInput.EnableVotingEndReminder {
			enableVotingEndReminder = true
		}
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

func (s *SubscribeService) SubscribeProposal(baseInput types.BasicInput[gqlmodels.SubscribeProposalInput]) (*gqlmodels.SubscribedProposalOutput, error) {
	// Implement subscription logic here
	return nil, nil
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
