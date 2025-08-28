package services

import (
	"github.com/ringecosystem/degov-apps/database"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/types"
	"gorm.io/gorm"
)

type SubscribeService struct {
	db *gorm.DB
}

func NewSubscribeService() *SubscribeService {
	return &SubscribeService{
		db: database.GetDB(),
	}
}

func (s *SubscribeService) SubscribeDao(baseInput types.BasicInput[gqlmodels.SubscribeDaoInput]) (*gqlmodels.SubscribedDaoOutput, error) {
	// Implement subscription logic here
	return nil, nil
}

func (s *SubscribeService) SubscribeProposal(baseInput types.BasicInput[gqlmodels.SubscribeProposalInput]) (*gqlmodels.SubscribedProposalOutput, error) {
	// Implement subscription logic here
	return nil, nil
}
