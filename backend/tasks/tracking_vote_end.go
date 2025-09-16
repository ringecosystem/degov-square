package tasks

import (
	"log/slog"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/internal/utils"
	"github.com/ringecosystem/degov-square/services"
	"github.com/ringecosystem/degov-square/types"
)

type TrackingVoteEndTask struct {
	daoService          *services.DaoService
	daoConfigService    *services.DaoConfigService
	notificationService *services.NotificationService
}

func NewTrackingVoteEndTask() *TrackingVoteEndTask {
	return &TrackingVoteEndTask{
		daoService:          services.NewDaoService(),
		daoConfigService:    services.NewDaoConfigService(),
		notificationService: services.NewNotificationService(),
	}
}

// Name returns the task name
func (t *TrackingVoteEndTask) Name() string {
	return "tracking-vote-end"
}

// Execute performs the DAO synchronization
func (t *TrackingVoteEndTask) Execute() error {
	return t.trackingVoteEnd()
}

func (t *TrackingVoteEndTask) trackingVoteEnd() error {
	daos, err := t.daoService.ListDaos(types.BasicInput[*types.ListDaosInput]{})
	if err != nil {
		slog.Error("Failed to list DAOs", "error", err)
		return err
	}

	for _, dao := range daos {
		daoConfig, err := t.daoConfigService.StandardConfig(dao.Code)
		if err != nil {
			slog.Error("Failed to get DAO config", "dao_code", dao.Code, "error", err)
			continue
		}

		indexer := internal.NewDegovIndexer(daoConfig.Indexer.Endpoint)

		proposals, err := indexer.QueryExpiringProposals()

		if err != nil {
			// return nil, fmt.Errorf("failed to query votes: %w", err)
			slog.Warn("Failed to query expiring proposals", "dao_code", dao.Code, "error", err)
			continue
		}

		notificationEvents := []dbmodels.NotificationEvent{}
		for _, proposal := range proposals {
			slog.Info(
				"Proposal is expiring soon",
				"dao_code", dao.Code,
				"proposal_id", proposal.ProposalID,
				"vote_end_time", proposal.VoteEndTimestamp,
			)
			existingEvent, _ := t.notificationService.InspectEventWithProposal(types.InspectNotificationEventInput{
				DaoCode:    dao.Code,
				ProposalID: proposal.ProposalID,
				Type:       dbmodels.SubscribeFeatureVoteEnd,
			})
			if existingEvent != nil {
				slog.Info("Existing notification event found", "event", existingEvent)
				continue
			}

			voteEndTime, err := utils.ParseTimestamp(proposal.VoteEndTimestamp)
			if err != nil {
				slog.Warn("Failed to parse VoteEndTimestamp", "proposal_id", proposal.ProposalID, "timestamp", proposal.VoteEndTimestamp, "error", err)
				continue
			}
			ne := dbmodels.NotificationEvent{
				ChainID:    int(dao.ChainID),
				DaoCode:    dao.Code,
				Type:       dbmodels.SubscribeFeatureVoteEnd,
				ProposalID: proposal.ProposalID,
				TimeEvent:  voteEndTime,
			}
			notificationEvents = append(notificationEvents, ne)
		}
		if err := t.notificationService.SaveEvents(notificationEvents); err != nil {
			slog.Warn("Failed to save notification events", "dao_code", dao.Code, "error", err)
		}
	}

	return nil
}
