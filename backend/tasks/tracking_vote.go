package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/internal"
	"github.com/ringecosystem/degov-apps/services"
	"github.com/ringecosystem/degov-apps/types"
)

type TrackingVoteTask struct {
	daoService       *services.DaoService
	proposalService  *services.ProposalService
	daoConfigService *services.DaoConfigService
}

func NewTrackingVoteTask() *TrackingVoteTask {
	return &TrackingVoteTask{
		daoService:       services.NewDaoService(),
		proposalService:  services.NewProposalService(),
		daoConfigService: services.NewDaoConfigService(),
	}
}

// Name returns the task name
func (t *TrackingVoteTask) Name() string {
	return "tracking-vote"
}

// Execute performs the DAO synchronization
func (t *TrackingVoteTask) Execute() error {
	return t.TrackingVote()
}

type trackingVoteInput struct {
	indexer   *internal.DegovIndexer
	daoConfig *types.DaoConfig
	dao       *gqlmodels.Dao
	proposal  *dbmodels.ProposalTracking
}

func (t *TrackingVoteTask) TrackingVote() error {
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
				slog.Error("Failed to track vote by proposal", "error", err, "dao", dao.Code, "proposal", proposal.ProposalId)
				continue
			}
			slog.Info("Tracked vote by proposal", "dao", dao.Code, "proposal", proposal.ProposalId)
		}
	}
	return nil
}

func (t *TrackingVoteTask) trackingVoteByProposal(input trackingVoteInput) error {
	indexer := input.indexer
	proposal := input.proposal
	lastOffsetVote := proposal.OffsetTrackingVote
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		votes, err := indexer.QueryVotesOffset(ctx, lastOffsetVote, proposal.ProposalId)
		cancel()

		if err != nil {
			return fmt.Errorf("failed to query votes: %w", err)
		}

		if len(votes) == 0 {
			slog.Info("No more votes found for this proposal", "dao_code", proposal.DaoCode, "proposal", proposal.ProposalId)
			break
		}

		for _, vote := range votes {

		}
	}
	return nil
}
