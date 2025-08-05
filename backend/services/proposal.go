package services

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	"github.com/ringecosystem/degov-apps/internal/utils"
	"github.com/ringecosystem/degov-apps/types"
)

type ProposalService struct {
	db *gorm.DB
}

func NewProposalService() *ProposalService {
	return &ProposalService{
		db: database.GetDB(),
	}
}

// StoreProposalTracking checks if proposal exists and creates it if not
// Returns true if proposal was newly created, false if it already existed
func (s *ProposalService) StoreProposalTracking(input types.ProposalTrackingInput) (bool, error) {
	// Check if proposal already exists
	var existingProposal dbmodels.ProposalTracking
	err := s.db.Where("proposal_id = ? AND dao_code = ?", input.ProposalID, input.DaoCode).First(&existingProposal).Error

	if err == nil {
		// Proposal already exists
		return false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Database error
		return false, err
	}

	// Proposal doesn't exist, create it
	newProposal := &dbmodels.ProposalTracking{
		ID:                utils.NextIDString(),
		DaoCode:           input.DaoCode,
		ProposalLink:      input.ProposalLink,
		ProposalId:        input.ProposalID,
		State:             dbmodels.ProposalStatePending,
		ProposalCreatedAt: input.ProposalCreatedAt,
		ProposalAtBlock:   input.ProposalAtBlock,
		CTime:             time.Now(),
	}

	if err := s.db.Create(newProposal).Error; err != nil {
		return false, err
	}

	return true, nil
}
