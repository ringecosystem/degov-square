package tasks

import (
	"log/slog"

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
		// if err := t.notificationService.CreateNotificationRecord(event); err != nil {
		// 	return err
		// }
		// t.subscribeService.ListSubscribedUser()
		if err := t.buildNotificationRecordByEvent(&event); err != nil {
			slog.Error("Failed to build notification record", "event_id", event.ID, "error", err)
		}
	}

	return nil
}

func (t *NotificationEventTask) buildNotificationRecordByEvent(event *dbmodels.NotificationEvent) error {
	// records := []dbmodels.NotificationRecord{}
	// t.subscribeService.ListSubscribedUser(types.ListSubscribeUserInput{
	// 	Feature: event.Feature,
	// 	// Strategies []string{string(event.Strategy)},
	// 	DaoCode:    event.DaoCode,
	// 	ProposalID: event.ProposalID,
	// 	TimeEvent:  event.TimeEvent,
	// 	Limit:      100,
	// 	Offset:     0,
	// })
	return nil
}
