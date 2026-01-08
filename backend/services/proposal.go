package services

import (
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/jinzhu/copier"
	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/internal/utils"
	"github.com/ringecosystem/degov-square/types"
)

type ProposalService struct {
	db                  *gorm.DB
	notificationService *NotificationService
}

func NewProposalService() *ProposalService {
	return &ProposalService{
		db:                  database.GetDB(),
		notificationService: NewNotificationService(),
	}
}

// StoreProposalTracking checks if proposal exists and creates it if not
// Returns true if proposal was newly created, false if it already existed
func (s *ProposalService) StoreProposalTracking(input types.ProposalTrackingInput) (bool, error) {
	// Check if proposal already exists
	var existingProposal dbmodels.ProposalTracking
	err := s.db.
		Where("proposal_id = ? AND dao_code = ?", input.ProposalID, input.DaoCode).
		First(&existingProposal).
		Error

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
		Title:             input.Title,
		ProposalLink:      input.ProposalLink,
		ProposalID:        input.ProposalID,
		State:             dbmodels.ProposalStateUnknown, // Default state
		ProposalCreatedAt: input.ProposalCreatedAt,
		ProposalAtBlock:   input.ProposalAtBlock,
		CTime:             time.Now(),
	}

	if err := s.db.Create(newProposal).Error; err != nil {
		return false, err
	}

	if err := s.notificationService.SaveEvent(dbmodels.NotificationEvent{
		ChainID:    newProposal.ChainId,
		DaoCode:    newProposal.DaoCode,
		Type:       dbmodels.SubscribeFeatureProposalNew,
		ProposalID: newProposal.ProposalID,
		TimeEvent:  newProposal.CTime,
	}); err != nil {
		slog.Warn("failed to save notification event for new proposal", "error", err, "proposal_id", newProposal.ProposalID, "dao_code", newProposal.DaoCode)
	}

	return true, nil
}

func (s *ProposalService) TrackingStateProposals(input types.TrackingStateProposalsInput) ([]*dbmodels.ProposalTracking, error) {
	var proposals []*dbmodels.ProposalTracking

	timesTrack := 10
	if input.TimesTrack != nil {
		timesTrack = *input.TimesTrack
	}

	// Query proposals with specific states, tracking limits, and time conditions
	err := s.db.Where(`dao_code = ?
		AND state IN ?
		AND times_track < ?
		AND (time_next_track IS NULL OR time_next_track <= ?)`,
		input.DaoCode,
		input.States,
		timesTrack,
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
			"state": newState,
			"utime": time.Now(),
		}).Error
}

// UpdateProposalTrackingError updates tracking info when processing fails
func (s *ProposalService) UpdateProposalTrackingError(proposalID, daoCode string, errorMessage string) error {
	// Get current proposal to build new message
	var proposal dbmodels.ProposalTracking
	err := s.db.Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).First(&proposal).Error
	if err != nil {
		return err
	}

	// Calculate next tracking time (exponential backoff based on attempts)
	nextTrackTime := time.Now()
	attempts := proposal.TimesTrack + 1
	if attempts <= 3 {
		nextTrackTime = nextTrackTime.Add(time.Hour) // 1 hour for first 3 attempts
	} else if attempts <= 6 {
		nextTrackTime = nextTrackTime.Add(2 * time.Hour) // 2 hours for attempts 4-6
	} else {
		nextTrackTime = nextTrackTime.Add(5 * time.Hour) // 5 hours for more attempts
	}

	// Build new message in format: [${time_next_track}] ${message}\n----\nOLD_MESSAGE
	newMessage := "[" + nextTrackTime.Format("2006-01-02 15:04:05") + "] " + errorMessage
	if proposal.Message != "" {
		newMessage += "\n----\n" + proposal.Message
	}

	return s.db.Model(&dbmodels.ProposalTracking{}).
		Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).
		Updates(map[string]interface{}{
			"times_track":     gorm.Expr("times_track + 1"),
			"time_next_track": nextTrackTime,
			"message":         newMessage,
			"utime":           time.Now(),
		}).Error
}

func (s *ProposalService) UpdateOffsetTrackingVote(proposalID, daoCode string, offset int) error {
	return s.db.Model(&dbmodels.ProposalTracking{}).
		Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).
		Updates(map[string]interface{}{
			"offset_tracking_vote": offset,
			"utime":                time.Now(),
		}).Error
}

// ProposalStateCount returns count of proposals by DAO and state for active DAOs
func (s *ProposalService) ProposalStateCount() ([]types.ProposalStateCountResult, error) {
	var results []types.ProposalStateCountResult

	err := s.db.Table("dgv_proposal_tracking as t").
		Select("t.dao_code, t.state, count(1) as total").
		Joins("left join dgv_dao as d on t.dao_code = d.code").
		Where("d.state = ?", "ACTIVE").
		Group("t.dao_code, t.state").
		Order("t.dao_code, t.state").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (s *ProposalService) InspectProposal(input types.InspectProposalInput) (*dbmodels.ProposalTracking, error) {
	var proposal dbmodels.ProposalTracking
	err := s.db.Table("dgv_proposal_tracking").
		Where("dao_code = ? AND proposal_id = ?", input.DaoCode, input.ProposalID).
		First(&proposal).Error
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

// FindByChainAndProposalID finds a proposal by chain ID and proposal ID
// Optional fulfilled filter can be applied
func (s *ProposalService) FindByChainAndProposalID(chainID int, proposalID string, fulfilled *int) (*dbmodels.ProposalTracking, error) {
	var proposal dbmodels.ProposalTracking
	query := s.db.Table("dgv_proposal_tracking").
		Where("chain_id = ? AND proposal_id = ?", chainID, proposalID)

	if fulfilled != nil {
		query = query.Where("fulfilled = ?", *fulfilled)
	}

	err := query.First(&proposal).Error
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

func (s *ProposalService) ConvertToGqlProposal(input *dbmodels.ProposalTracking) *gqlmodels.Proposal {
	gqlProposal := gqlmodels.Proposal{}
	copier.Copy(&gqlProposal, input)
	return &gqlProposal
}

func (s *ProposalService) SummaryProposalStates(input gqlmodels.SummaryProposalStatesInput) ([]*gqlmodels.SummaryProposalStates, error) {
	var results []*gqlmodels.SummaryProposalStates

	err := s.db.Table("dgv_proposal_tracking").
		Select("state, count(1) as count").
		Where("dao_code = ?", input.DaoCode).
		Group("state").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

// ListUnfulfilledProposals returns proposals that need AI fulfill processing
// Only returns proposals that are Active, not fulfilled, not errored, and under retry limit
// If supportedDAOs is provided (non-nil), only proposals from those DAOs are returned
func (s *ProposalService) ListUnfulfilledProposals(supportedDAOs []string) ([]*dbmodels.ProposalTracking, error) {
	var proposals []*dbmodels.ProposalTracking

	// slog.Info("[proposal-service] ListUnfulfilledProposals called",
	// 	"supportedDAOs", supportedDAOs,
	// 	"expected_state", dbmodels.ProposalStateActive)
	// // First, let's check what proposals exist for these DAOs (for debugging)
	// var allProposals []*dbmodels.ProposalTracking
	// debugQuery := s.db.Model(&dbmodels.ProposalTracking{})
	// if len(supportedDAOs) > 0 {
	// 	debugQuery = debugQuery.Where("dao_code IN ?", supportedDAOs)
	// }
	// debugQuery.Find(&allProposals)
	// slog.Info("[proposal-service] All proposals in DB for DAOs",
	// 	"count", len(allProposals),
	// 	"supportedDAOs", supportedDAOs)
	// for i, p := range allProposals {
	// 	slog.Info("[proposal-service] Existing proposal",
	// 		"index", i,
	// 		"proposal_id", p.ProposalID,
	// 		"dao_code", p.DaoCode,
	// 		"state", p.State,
	// 		"fulfilled", p.Fulfilled,
	// 		"fulfill_errored", p.FulfillErrored,
	// 		"times_fulfill", p.TimesFulfill)
	// }

	query := s.db.Where(`
		state = ?
		AND fulfilled = 0
		AND fulfill_errored = 0
		AND times_fulfill <= 4
	`, dbmodels.ProposalStateActive)

	// Filter by supported DAOs if specified
	if len(supportedDAOs) > 0 {
		query = query.Where("dao_code IN ?", supportedDAOs)
	}

	err := query.Order("ctime asc, times_fulfill asc").
		Find(&proposals).Error

	if err != nil {
		slog.Error("[proposal-service] Query error", "error", err)
		return nil, err
	}

	slog.Info("[proposal-service] ListUnfulfilledProposals result", "count", len(proposals))
	return proposals, nil
}

// UpdateProposalFulfilled marks a proposal as fulfilled with AI explanation
func (s *ProposalService) UpdateProposalFulfilled(proposalID, daoCode string, fulfilledExplain string) error {
	now := time.Now()
	return s.db.Model(&dbmodels.ProposalTracking{}).
		Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).
		Updates(map[string]interface{}{
			"fulfilled":         1,
			"fulfilled_explain": fulfilledExplain,
			"fulfilled_at":      now,
			"utime":             now,
		}).Error
}

// UpdateFulfillError updates fulfill tracking info when processing fails
func (s *ProposalService) UpdateFulfillError(proposalID, daoCode string, errorMessage string) error {
	// Get current proposal
	var proposal dbmodels.ProposalTracking
	err := s.db.Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).First(&proposal).Error
	if err != nil {
		return err
	}

	timesFulfill := proposal.TimesFulfill + 1
	fulfillErrored := 0
	if timesFulfill > 3 {
		fulfillErrored = 1
	}

	// Build message with timestamp
	newMessage := "[" + time.Now().Format("2006-01-02 15:04:05") + "] [fulfill] " + errorMessage
	if proposal.Message != "" {
		newMessage += "\n----\n" + proposal.Message
	}

	return s.db.Model(&dbmodels.ProposalTracking{}).
		Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).
		Updates(map[string]interface{}{
			"times_fulfill":   timesFulfill,
			"fulfill_errored": fulfillErrored,
			"message":         newMessage,
			"utime":           time.Now(),
		}).Error
}

// MarkFulfillExpired marks a proposal as expired for fulfill processing
// This is called when the voting period has ended and the proposal can no longer be voted on
func (s *ProposalService) MarkFulfillExpired(proposalID, daoCode string) error {
	newMessage := "[" + time.Now().Format("2006-01-02 15:04:05") + "] [fulfill] Voting period expired, skipping"

	// Get current proposal to prepend message
	var proposal dbmodels.ProposalTracking
	if err := s.db.Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).First(&proposal).Error; err != nil {
		return err
	}

	finalMessage := newMessage
	if proposal.Message != "" {
		finalMessage = newMessage + "\n----\n" + proposal.Message
	}

	return s.db.Model(&dbmodels.ProposalTracking{}).
		Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).
		Updates(map[string]interface{}{
			"fulfill_errored": 1,
			"message":         finalMessage,
			"utime":           time.Now(),
		}).Error
}

// MarkFulfillNoDelegators marks a proposal as skipped due to no delegators
// This is called when the agent has no delegators other than itself
func (s *ProposalService) MarkFulfillNoDelegators(proposalID, daoCode string) error {
	newMessage := "[" + time.Now().Format("2006-01-02 15:04:05") + "] [fulfill] No delegators found for agent, skipping"

	// Get current proposal to prepend message
	var proposal dbmodels.ProposalTracking
	if err := s.db.Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).First(&proposal).Error; err != nil {
		return err
	}

	finalMessage := newMessage
	if proposal.Message != "" {
		finalMessage = newMessage + "\n----\n" + proposal.Message
	}

	return s.db.Model(&dbmodels.ProposalTracking{}).
		Where("proposal_id = ? AND dao_code = ?", proposalID, daoCode).
		Updates(map[string]interface{}{
			"fulfill_errored": 1,
			"message":         finalMessage,
			"utime":           time.Now(),
		}).Error
}
