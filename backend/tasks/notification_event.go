package tasks

import (
	"fmt"
	"log/slog"
	"time"

	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	"github.com/ringecosystem/degov-apps/services"
	"github.com/ringecosystem/degov-apps/types"
)

type NotificationEventTask struct {
	daoService          *services.DaoService
	notificationService *services.NotificationService
	subscribeService    *services.SubscribeService
}

func NewNotificationEventTask() *NotificationEventTask {
	return &NotificationEventTask{
		daoService:          services.NewDaoService(),
		notificationService: services.NewNotificationService(),
		subscribeService:    services.NewSubscribeService(),
	}
}

func (t *NotificationEventTask) Name() string {
	return "notification-event"
}

func (t *NotificationEventTask) Execute() error {
	return t.buildNotificationRecord()
}

func (t *NotificationEventTask) buildNotificationRecord() error {
	events, err := t.notificationService.ListLimitEvents(types.ListLimitEventsInput{
		Limit: 10,
	})
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := t.notificationService.UpdateEventState(types.UpdateEventStateInput{
			ID:    event.ID,
			State: dbmodels.NotificationEventStateProgress,
		}); err != nil {
			slog.Error("Failed to update event state to progress", "event_id", event.ID, "error", err)
			continue
		}

		if err := t.buildNotificationRecordByEvent(&event); err != nil {
			slog.Error("Failed to build notification record", "event_id", event.ID, "error", err)
			if err := t.notificationService.UpdateEventRetryTimes(types.UpdateEventRetryTimes{
				ID:         event.ID,
				TimesRetry: event.TimesRetry + 1,
				Message:    fmt.Sprintf("%s \n\nFailed to build notification record: %s", *event.Message, err.Error()),
			}); err != nil {
				slog.Error("Failed to update event retry times", "event_id", event.ID, "error", err)
			}
			continue
		}

		if err := t.notificationService.UpdateEventState(types.UpdateEventStateInput{
			ID:    event.ID,
			State: dbmodels.NotificationEventStateCompleted,
		}); err != nil {
			slog.Error("Failed to update event state to completed", "event_id", event.ID, "error", err)
			continue
		}
	}

	return nil
}

func (t *NotificationEventTask) buildNotificationRecordByEvent(event *dbmodels.NotificationEvent) error {
	var (
		offset     = 0
		limit      = 100
		recordsBuf = make([]dbmodels.NotificationRecord, 0, 256)
		batchSize  = 200
	)
	for {
		subscribedUsers, err := t.subscribeService.ListSubscribedUser(types.ListSubscribeUserInput{
			Feature:    event.Type,
			Strategies: t.allowStrategies(event.Type),
			DaoCode:    event.DaoCode,
			ProposalID: &event.ProposalID,
			TimeEvent:  &event.TimeEvent,
			Limit:      limit,
			Offset:     offset,
		})
		if err != nil {
			return err
		}
		for _, user := range subscribedUsers {
			rec := dbmodels.NotificationRecord{
				Code:        event.ID + "_" + user.UserID,
				EventID:     event.ID,
				ChainID:     event.ChainID,
				DaoCode:     event.DaoCode,
				Type:        event.Type,
				ProposalID:  event.ProposalID,
				VoteID:      event.VoteID,
				UserID:      user.UserID,
				UserAddress: user.UserAddress,
				State:       dbmodels.NotificationRecordStatePending,
				TimesRetry:  0,
				CTime:       time.Now(),
			}
			recordsBuf = append(recordsBuf, rec)

			if len(recordsBuf) >= batchSize {
				if err := t.notificationService.StoreRecords(recordsBuf); err != nil {
					return fmt.Errorf("failed to store notification records: %w", err)
				}
				recordsBuf = recordsBuf[:0] // reset buffer
			}
		}
		if len(subscribedUsers) < limit {
			break
		}
		offset += limit
	}

	// flush remaining buffered records
	if len(recordsBuf) > 0 {
		if err := t.notificationService.StoreRecords(recordsBuf); err != nil {
			return fmt.Errorf("failed to store notification records: %w", err)
		}
	}
	return nil
}

func (t *NotificationEventTask) allowStrategies(feature dbmodels.SubscribeFeatureName) []string {
	switch feature {
	case dbmodels.SubscribeFeatureProposalNew:
		return []string{"true"}
	case dbmodels.SubscribeFeatureProposalStateChanged:
		return []string{"true"}
	case dbmodels.SubscribeFeatureVoteEmitted:
		return []string{"true"}
	case dbmodels.SubscribeFeatureVoteEnd:
		return []string{"true"}
	default:
		return nil
	}
}
