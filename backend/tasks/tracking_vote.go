package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/internal/utils"
	"github.com/ringecosystem/degov-square/services"
	"github.com/ringecosystem/degov-square/types"
)

type TrackingVoteTask struct {
	daoService          *services.DaoService
	proposalService     *services.ProposalService
	daoConfigService    *services.DaoConfigService
	notificationService *services.NotificationService
}

func NewTrackingVoteTask() *TrackingVoteTask {
	return &TrackingVoteTask{
		daoService:          services.NewDaoService(),
		proposalService:     services.NewProposalService(),
		daoConfigService:    services.NewDaoConfigService(),
		notificationService: services.NewNotificationService(),
	}
}

// Name returns the task name
func (t *TrackingVoteTask) Name() string {
	return "tracking-vote"
}

// Execute performs the DAO synchronization
func (t *TrackingVoteTask) Execute() error {
	return t.trackingVote()
}

type trackingVoteInput struct {
	indexer   *internal.DegovIndexer
	daoConfig *types.DaoConfig
	dao       *gqlmodels.Dao
	proposal  *dbmodels.ProposalTracking
}

func (t *TrackingVoteTask) trackingVote() error {
	daos, err := t.daoService.ListDaos(types.BasicInput[*types.ListDaosInput]{})
	if err != nil {
		slog.Error("Failed to list DAOs", "error", err)
		return err
	}

	for _, dao := range daos {
		// Get DAO config from DaoConfigService by DaoCode
		daoConfig, err := t.daoConfigService.StandardConfig(dao.Code)
		if err != nil {
			slog.Error("Failed to get DAO config", "dao_code", dao.Code, "error", err)
			continue
		}

		timesTrack := 100
		proposals, err := t.proposalService.TrackingStateProposals(types.TrackingStateProposalsInput{
			DaoCode:    dao.Code,
			TimesTrack: &timesTrack,
			States: []dbmodels.ProposalState{
				dbmodels.ProposalStateActive,
			},
		})
		if err != nil {
			slog.Error("Failed to track vote, reasoning failed to fetch proposals", "error", err)
			return err
		}
		indexer := internal.NewDegovIndexer(daoConfig.Indexer.Endpoint)
		for _, proposal := range proposals {
			if err := t.trackingVoteByProposal(trackingVoteInput{
				indexer:   indexer,
				daoConfig: daoConfig,
				dao:       dao,
				proposal:  proposal,
			}); err != nil {
				slog.Error("Failed to track vote by proposal", "error", err, "dao", dao.Code, "proposal", proposal.ProposalID)
				continue
			}
			slog.Info("Tracked vote by proposal", "dao", dao.Code, "proposal", proposal.ProposalID)
		}
	}
	return nil
}

func (t *TrackingVoteTask) trackingVoteByProposal(input trackingVoteInput) error {
	// 1. Fetch and process all new votes at once
	processedVotes, err := t.fetchAllAndProcessVotes(input)
	if err != nil {
		return err // error already wrapped internally
	}

	if len(processedVotes) == 0 {
		slog.Info("No new votes to process", "dao_code", input.proposal.DaoCode, "proposal", input.proposal.ProposalID)
		return nil
	}

	// 2. Page through subscribed users using the earliest time and generate notifications
	return t.generateAndStoreNotificationEvents(input.proposal, processedVotes)
}

func (t *TrackingVoteTask) fetchAllAndProcessVotes(input trackingVoteInput) ([]processedVote, error) {
	var (
		indexer        = input.indexer
		proposal       = input.proposal
		lastOffsetVote = proposal.OffsetTrackingVote
		processedVotes = make([]processedVote, 0)
	)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		votes, err := indexer.QueryVotesOffset(ctx, lastOffsetVote, proposal.ProposalID)
		cancel()

		if err != nil {
			return nil, fmt.Errorf("failed to query votes: %w", err)
		}
		if len(votes) == 0 {
			break
		}

		for _, v := range votes {
			ts, err := utils.ParseTimestamp(v.BlockTimestamp)
			if err != nil {
				slog.Warn("Skipping vote due to unparsable timestamp", "vote_id", v.ID, "timestamp", v.BlockTimestamp, "error", err)
				continue
			}
			processedVotes = append(processedVotes, processedVote{Vote: v, Timestamp: ts})
		}

		lastOffsetVote += len(votes)
		if err := t.proposalService.UpdateOffsetTrackingVote(proposal.ProposalID, proposal.DaoCode, lastOffsetVote); err != nil {
			return nil, fmt.Errorf("failed to update offset tracking vote: %w", err)
		}
	}
	return processedVotes, nil
}

func (t *TrackingVoteTask) generateAndStoreNotificationEvents(proposal *dbmodels.ProposalTracking, processedVotes []processedVote) error {
	notificationEvents := []dbmodels.NotificationEvent{}
	for _, vote := range processedVotes {
		ne := dbmodels.NotificationEvent{
			ChainID:    proposal.ChainId,
			DaoCode:    proposal.DaoCode,
			Type:       dbmodels.SubscribeFeatureVoteEmitted,
			ProposalID: proposal.ProposalID,
			VoteID:     &vote.Vote.ID,
			TimeEvent:  vote.Timestamp,
		}
		notificationEvents = append(notificationEvents, ne)
	}
	return t.notificationService.SaveEvents(notificationEvents)
}

type processedVote struct {
	Vote      internal.VoteCast
	Timestamp time.Time
}
