package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/services"
	"github.com/ringecosystem/degov-square/types"
)

type TrackingProposalTask struct {
	daoService          *services.DaoService
	daoConfigService    *services.DaoConfigService
	proposalService     *services.ProposalService
	chipService         *services.DaoChipService
	notificationService *services.NotificationService
}

func NewTrackingProposalTask() *TrackingProposalTask {
	return &TrackingProposalTask{
		daoService:          services.NewDaoService(),
		daoConfigService:    services.NewDaoConfigService(),
		proposalService:     services.NewProposalService(),
		chipService:         services.NewDaoChipService(),
		notificationService: services.NewNotificationService(),
	}
}

// Name returns the task name
func (t *TrackingProposalTask) Name() string {
	return "tracking-proposal"
}

// Execute performs the DAO synchronization
func (t *TrackingProposalTask) Execute() error {
	return t.trackingProposal()
}

// TrackingProposal tracks proposals for DAOs
func (t *TrackingProposalTask) trackingProposal() error {
	// Get all DAOs from DaoService.ListDaos
	daos, err := t.daoService.ListDaos(types.BasicInput[*types.ListDaosInput]{})
	if err != nil {
		slog.Error("Failed to list DAOs", "error", err)
		return err
	}

	slog.Info("Found DAOs for proposal tracking", "count", len(daos))

	// Iterate through each DAO and get its config from DaoConfigService
	for _, dao := range daos {
		daoConfig, err := t.daoConfigService.StandardConfig(dao.Code)
		if err != nil {
			slog.Error("Failed to get DAO config", "dao_code", dao.Code, "error", err)
			continue
		}

		slog.Info(
			"Processing DAO",
			"dao_code", dao.Code,
			"dao_name", daoConfig.Name,
			"indexer_endpoint", daoConfig.Indexer.Endpoint,
		)

		if err := t.storeProposals(dao, daoConfig); err != nil {
			slog.Error("Failed to process proposal tracking", "dao_code", dao.Code, "error", err)
			continue
		}
		if err := t.updateProposalsStates(dao, daoConfig); err != nil {
			slog.Error("Failed to update proposal state", "dao_code", dao.Code, "error", err)
			continue
		}
	}
	if err := t.updateDaoChips(); err != nil {
		slog.Warn("Failed to update DAO chips", "error", err)
		return err
	}

	return nil
}

func (t *TrackingProposalTask) storeProposals(dao *gqlmodels.Dao, daoConfig *types.DaoConfig) error {
	indexer := internal.NewDegovIndexer(daoConfig.Indexer.Endpoint)
	scope := internal.ProposalScope{
		ChainID:         daoConfig.Chain.ID,
		DaoCode:         dao.Code,
		GovernorAddress: daoConfig.Contracts.Governor,
	}

	lastTrackedBlockNumber, lastTrackedProposalID, err := t.daoService.GetLastTrackedProposalCursor(dao.Code)
	if err != nil {
		return fmt.Errorf("failed to get last tracked proposal cursor: %w", err)
	}

	lastTrackedBlockNumber, lastTrackedProposalID, err = t.bootstrapProposalCursor(indexer, scope, dao, lastTrackedBlockNumber, lastTrackedProposalID)
	if err != nil {
		return err
	}

	initialBlockNumber := lastTrackedBlockNumber
	initialProposalID := lastTrackedProposalID

	slog.Info("Starting proposal tracking",
		"dao_code", dao.Code,
		"after_block_number", lastTrackedBlockNumber,
		"after_proposal_id", lastTrackedProposalID)

	for {
		proposals, err := indexer.QueryProposalsByBlockNumber(scope, lastTrackedBlockNumber, lastTrackedProposalID)
		if err != nil {
			return fmt.Errorf("failed to query proposals: %w", err)
		}

		if len(proposals) == 0 {
			slog.Info("No more proposals found", "dao_code", dao.Code)
			break
		}

		slog.Info("Found proposals", "dao_code", dao.Code, "count", len(proposals))

		var batchErr error
		for _, proposal := range proposals {
			if proposal.ID == "" {
				slog.Error("Proposal missing indexer id",
					"dao_code", dao.Code,
					"proposal_id", proposal.ProposalID)
				batchErr = fmt.Errorf("proposal %s missing indexer id", proposal.ProposalID)
				break
			}

			blockNumber, err := strconv.ParseInt(proposal.BlockNumber, 10, 64)
			if err != nil {
				slog.Error("Failed to parse block number",
					"dao_code", dao.Code,
					"proposal_id", proposal.ProposalID,
					"block_number", proposal.BlockNumber,
					"error", err)
				// Stop processing this batch; retry next time to avoid skipping proposals
				batchErr = fmt.Errorf("failed to parse proposal block number: %w", err)
				break
			}

			var proposalCreatedAt *time.Time
			if proposal.BlockTimestamp != "" {
				if timestamp, err := strconv.ParseInt(proposal.BlockTimestamp, 10, 64); err == nil {
					createdAt := time.Unix(timestamp/1000, (timestamp%1000)*1000000)
					proposalCreatedAt = &createdAt
				}
			}

			proposalLink := fmt.Sprintf("%s/proposal/%s", daoConfig.SiteURL, proposal.ProposalID)

			input := types.ProposalTrackingInput{
				DaoCode:           dao.Code,
				ChainId:           daoConfig.Chain.ID,
				Title:             proposal.Title,
				ProposalLink:      proposalLink,
				ProposalID:        proposal.ProposalID,
				ProposalCreatedAt: proposalCreatedAt,
				ProposalAtBlock:   int(blockNumber),
			}

			created, err := t.proposalService.StoreProposalTracking(input)
			if err != nil {
				slog.Error("Failed to store proposal tracking",
					"dao_code", dao.Code,
					"proposal_id", proposal.ProposalID,
					"error", err)
				// Stop processing this batch; do not advance cursor, retry next time
				batchErr = fmt.Errorf("failed to store proposal tracking: %w", err)
				break
			}

			if created {
				slog.Info("Inserted new proposal tracking",
					"dao_code", dao.Code,
					"proposal_id", proposal.ProposalID,
					"block_number", blockNumber,
					"proposal_link", proposalLink)
			} else {
				slog.Debug("Proposal already exists, skipping",
					"dao_code", dao.Code,
					"proposal_id", proposal.ProposalID)
			}

			// Only advance cursor after successful store
			if blockNumber > lastTrackedBlockNumber ||
				(blockNumber == lastTrackedBlockNumber && proposal.ID > lastTrackedProposalID) {
				lastTrackedBlockNumber = blockNumber
				lastTrackedProposalID = proposal.ID
			}
		}

		if lastTrackedBlockNumber != initialBlockNumber || lastTrackedProposalID != initialProposalID {
			if err := t.daoService.UpdateDaoLastTrackedProposalCursor(dao.Code, lastTrackedBlockNumber, lastTrackedProposalID); err != nil {
				return fmt.Errorf("failed to update last tracked proposal cursor: %w", err)
			}
			slog.Info("Updated last tracked proposal cursor",
				"dao_code", dao.Code,
				"old_block", initialBlockNumber,
				"old_proposal_id", initialProposalID,
				"new_block", lastTrackedBlockNumber,
				"new_proposal_id", lastTrackedProposalID)
			initialBlockNumber = lastTrackedBlockNumber
			initialProposalID = lastTrackedProposalID
		}

		if batchErr != nil {
			return batchErr
		}
	}

	return nil
}

func (t *TrackingProposalTask) bootstrapProposalCursor(indexer *internal.DegovIndexer, scope internal.ProposalScope, dao *gqlmodels.Dao, lastTrackedBlockNumber int64, lastTrackedProposalID string) (int64, string, error) {
	if lastTrackedProposalID != "" || dao.OffsetTrackingProposal <= 0 {
		return lastTrackedBlockNumber, lastTrackedProposalID, nil
	}

	proposals, err := indexer.QueryProposalsOffset(scope, int(dao.OffsetTrackingProposal)-1)
	if err != nil {
		return 0, "", fmt.Errorf("failed to bootstrap proposal cursor from offset: %w", err)
	}
	if len(proposals) == 0 {
		slog.Warn("Failed to bootstrap proposal cursor from offset",
			"dao_code", dao.Code,
			"offset_tracking_proposal", dao.OffsetTrackingProposal)
		return lastTrackedBlockNumber, lastTrackedProposalID, nil
	}

	blockNumber, err := strconv.ParseInt(proposals[0].BlockNumber, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("failed to parse bootstrap proposal block number: %w", err)
	}

	if blockNumber != lastTrackedBlockNumber {
		if err := t.daoService.UpdateDaoLastTrackedProposalCursor(dao.Code, blockNumber, ""); err != nil {
			return 0, "", fmt.Errorf("failed to update bootstrapped proposal cursor: %w", err)
		}
	}

	slog.Info("Bootstrapped proposal cursor from offset",
		"dao_code", dao.Code,
		"offset_tracking_proposal", dao.OffsetTrackingProposal,
		"last_tracked_block_number", blockNumber)

	return blockNumber, "", nil
}

func (t *TrackingProposalTask) updateProposalsStates(dao *gqlmodels.Dao, daoConfig *types.DaoConfig) error {
	proposals, err := t.proposalService.TrackingStateProposals(types.TrackingStateProposalsInput{
		DaoCode: dao.Code,
		States: []dbmodels.ProposalState{
			dbmodels.ProposalStateUnknown,
			dbmodels.ProposalStatePending,
			dbmodels.ProposalStateActive,
			dbmodels.ProposalStateSucceeded,
			dbmodels.ProposalStateQueued,
		},
	})
	if err != nil {
		slog.Error("Failed to get tracking state proposals",
			"dao_code", dao.Code,
			"error", err)
		return err
	}

	if len(proposals) == 0 {
		slog.Info("No proposals to update state for", "dao_code", dao.Code)
		return nil
	}

	governorAddress := daoConfig.Contracts.Governor
	if governorAddress == "" {
		slog.Warn("No governor contract address configured", "dao_code", dao.Code)
		return nil
	}

	// Get RPC URL
	rpcURL := internal.GetRPCURL(daoConfig.Chain.RPCs, daoConfig.Chain.ID)
	if rpcURL == "" {
		slog.Warn("No RPC URL available for chain", "dao_code", dao.Code, "chain_id", daoConfig.Chain.ID)
		return nil
	}

	// Create Governor contract client
	governorContract, err := internal.NewGovernorContract(rpcURL)
	if err != nil {
		slog.Error("Failed to create Governor contract client", "dao_code", dao.Code, "rpc_url", rpcURL, "error", err)
		return err
	}
	defer governorContract.Close()

	slog.Info("Updating proposal states",
		"dao_code", dao.Code,
		"count", len(proposals),
		"governor_address", governorAddress,
		"rpc_url", rpcURL)

	// Process each proposal individually
	for _, proposal := range proposals {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		// Get proposal state from contract
		newState, err := governorContract.GetProposalState(ctx, governorAddress, proposal.ProposalID)
		cancel()

		if err != nil {
			// Update tracking info with error
			if updateErr := t.proposalService.UpdateProposalTrackingError(proposal.ProposalID, dao.Code, err.Error()); updateErr != nil {
				slog.Error("Failed to update proposal tracking error",
					"dao_code", dao.Code,
					"proposal_id", proposal.ProposalID,
					"original_error", err,
					"update_error", updateErr)
			} else {
				slog.Warn("Updated proposal tracking with error",
					"dao_code", dao.Code,
					"proposal_id", proposal.ProposalID,
					"error", err)
			}
			continue
		}

		if err := t.proposalService.ResetProposalTrackingStatus(proposal.ProposalID, dao.Code); err != nil {
			slog.Warn("Failed to reset proposal tracking retry metadata",
				"dao_code", dao.Code,
				"proposal_id", proposal.ProposalID,
				"error", err)
		}

		// Check if state has changed
		if newState != proposal.State {
			// Update proposal state in database
			if err := t.proposalService.UpdateProposalState(proposal.ProposalID, dao.Code, newState); err != nil {
				// Update tracking info with error
				if updateErr := t.proposalService.UpdateProposalTrackingError(proposal.ProposalID, dao.Code, err.Error()); updateErr != nil {
					slog.Error("Failed to update proposal tracking error after state update failure",
						"dao_code", dao.Code,
						"proposal_id", proposal.ProposalID,
						"original_error", err,
						"update_error", updateErr)
				}
				continue
			}

			payload := "{\"old_state\": \"" + string(proposal.State) + "\", \"new_state\": \"" + string(newState) + "\"}"
			if proposal.State == dbmodels.ProposalStateUnknown {
				payload = "{\"new_state\": \"" + string(newState) + "\"}"
			}
			if err := t.notificationService.SaveEvent(dbmodels.NotificationEvent{
				ChainID:    proposal.ChainId,
				DaoCode:    proposal.DaoCode,
				Type:       dbmodels.SubscribeFeatureProposalStateChanged,
				ProposalID: proposal.ProposalID,
				TimeEvent:  proposal.CTime,
				Payload:    &payload,
			}); err != nil {
				slog.Error("Failed to save state change notification event",
					"dao_code", dao.Code,
					"proposal_id", proposal.ProposalID,
					"error", err,
				)
				continue
			}

			slog.Info("Updated proposal state",
				"dao_code", dao.Code,
				"proposal_id", proposal.ProposalID,
				"old_state", proposal.State,
				"new_state", newState)
		}
	}

	return nil
}

func (t *TrackingProposalTask) updateDaoChips() error {
	// Get proposal state counts for all active DAOs
	counts, err := t.proposalService.ProposalStateCount()
	if err != nil {
		slog.Error("Failed to get proposal state counts", "error", err)
		return err
	}

	slog.Info("Retrieved proposal state counts", "total_records", len(counts))

	// Store metrics state chips using DaoChipService
	storeInput := types.StoreDaoChipMetricsStateInput{
		MetricsStates: counts,
	}

	if err := t.chipService.StoreChipMetricsState(storeInput); err != nil {
		slog.Error("Failed to store chip metrics state", "error", err)
		return err
	}

	slog.Info("Successfully updated DAO chips", "metrics_count", len(counts))
	return nil
}
