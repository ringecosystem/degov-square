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
		ChainId:           input.ChainId,
		ProposalLink:      input.ProposalLink,
		ProposalId:        input.ProposalID,
		State:             dbmodels.ProposalStateUnknown, // Default state
		ProposalCreatedAt: input.ProposalCreatedAt,
		ProposalAtBlock:   input.ProposalAtBlock,
		CTime:             time.Now(),
	}

	if err := s.db.Create(newProposal).Error; err != nil {
		return false, err
	}

	return true, nil
}

func (s *ProposalService) TrackingStateProposals(input types.TrackingStateProposalsInput) ([]*dbmodels.ProposalTracking, error) {
	var proposals []*dbmodels.ProposalTracking

	// // Define the states we want to track
	// trackingStates := []dbmodels.ProposalState{
	// 	dbmodels.ProposalStatePending,
	// 	dbmodels.ProposalStateActive,
	// 	dbmodels.ProposalStateSucceeded,
	// 	dbmodels.ProposalStateQueued,
	// }

	// Query proposals with specific states, tracking limits, and time conditions
	err := s.db.Where(`dao_code = ?
		AND state IN ?
		AND times_track < ?
		AND (time_next_track IS NULL OR time_next_track <= ?)`,
		input.DaoCode,
		input.States,
		10,
		time.Now(),
	).
		Order("proposal_created_at asc").
		Find(&proposals).Error

	if err != nil {
		return nil, err
	}
	return proposals, nil
}

// UpdateProposalState updates the state of a proposal
func (s *ProposalService) UpdateProposalState(proposalID, daoCode string, newState dbmodels.ProposalState) error {
	return s.db.Model(&dbmodels.ProposalTracking{}).
		Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).
		Updates(map[string]interface{}{
			"state":           newState,
			"times_track":     gorm.Expr("times_track + 1"),
			"time_next_track": time.Now().Add(time.Hour), // Next check in 1 hour
			"utime":           time.Now(),
		}).Error
}
