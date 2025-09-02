package services

import (
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	"github.com/ringecosystem/degov-apps/types"
)

type TemplateService struct {
	daoService      *DaoService
	proposalService *ProposalService
}

func NewTemplateService() *TemplateService {
	return &TemplateService{
		daoService:      NewDaoService(),
		proposalService: NewProposalService(),
	}
}

func (s *TemplateService) GenerateTemplateByNotificationRecord(record *dbmodels.NotificationRecord) (string, error) {
	dao, err := s.daoService.Inspect(types.BasicInput[string]{
		User:  nil,
		Input: record.DaoCode,
	})
	if err != nil {
		return "", err
	}

	proposal, err := s.proposalService.InspectProposal(types.InpspectProposalInput{
		DaoCode:    record.DaoCode,
		ProposalID: record.ProposalID,
	})
	if err != nil {
		return "", err
	}

	switch record.Type {
	case dbmodels.SubscribeFeatureProposalNew:

		break
	case dbmodels.SubscribeFeatureProposalStateChanged:
		break
	case dbmodels.SubscribeFeatureVoteEnd:
		break
	case dbmodels.SubscribeFeatureVoteEmitted:
		break
	}
	return "", nil
}
