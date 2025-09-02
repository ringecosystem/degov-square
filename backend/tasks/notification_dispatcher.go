package tasks

import (
	"log/slog"

	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	"github.com/ringecosystem/degov-apps/services"
	"github.com/ringecosystem/degov-apps/types"
)

type NotificationDispatcherTask struct {
	notificationService    *services.NotificationService
	templateService        *services.TemplateService
	notifierService        *services.NotifierService
	userInteractionService *services.UserInteractionService
}

func NewNotificationDispatcherTask() *NotificationDispatcherTask {
	return &NotificationDispatcherTask{
		notificationService:    services.NewNotificationService(),
		templateService:        services.NewTemplateService(),
		notifierService:        services.NewNotifierService(),
		userInteractionService: services.NewUserInteractionService(),
	}
}

func (t *NotificationDispatcherTask) Name() string {
	return "notification-dispatcher"
}

func (t *NotificationDispatcherTask) Execute() error {
	return t.dispatcherNotificationRecord()
}

func (t *NotificationDispatcherTask) dispatcherNotificationRecord() error {
	records, err := t.notificationService.ListLimitRecords(types.ListLimitRecordsInput{
		Limit: 10,
	})
	if err != nil {
		return err
	}
	for _, record := range records {
		channels, err := t.userInteractionService.ListChannel(types.BasicInput[types.ListChannelInput]{
			User: &types.UserSessInfo{
				Id: record.UserID,
			},
			Input: types.ListChannelInput{
				Verified: true,
			},
		})
		if err != nil {
			slog.Error("Failed to list user channels", "user_id", record.UserID, "error", err)
			t.notificationService.UpdateRecordRetryTimes(types.UpdateRecordRetryTimes{
				ID:         record.ID,
				TimesRetry: record.TimesRetry + 1,
				Message:    *record.Message + "\n\nFailed to list user channels: " + err.Error(),
			})
			continue
		}

		if err := t.dispatchNotificationRecordByRecord(&record, channels); err != nil {
			slog.Error("Failed to dispatch notification record", "record_id", record.ID, "error", err)

			timesRetry := record.TimesRetry + 1
			if err := t.notificationService.UpdateEventRetryTimes(types.UpdateEventRetryTimes{
				ID:         record.ID,
				TimesRetry: timesRetry,
				Message:    *record.Message + "\n\nFailed to build notification record: " + err.Error(),
			}); err != nil {
				slog.Error("Failed to update record retry times", "record_id", record.ID, "error", err)
			}

			if timesRetry > 4 {
				if err := t.notificationService.UpdateRecordState(types.UpdateRecordStateInput{
					ID:    record.ID,
					State: dbmodels.NotificationRecordStateSentFail,
				}); err != nil {
					slog.Error("Failed to update record state to failed", "record_id", record.ID, "error", err)
				}
			}
			continue
		}

		if err := t.notificationService.UpdateRecordState(types.UpdateRecordStateInput{
			ID:    record.ID,
			State: dbmodels.NotificationRecordStateSentOk,
		}); err != nil {
			slog.Error("Failed to update record state to send_ok", "record_id", record.ID, "error", err)
			continue
		}
	}
	return nil
}

func (t *NotificationDispatcherTask) dispatchNotificationRecordByRecord(record *dbmodels.NotificationRecord, channels []dbmodels.NotificationChannel) error {
	templateOutput, err := t.templateService.GenerateTemplateByNotificationRecord(record)
	if err != nil {
		return err
	}
	slog.Debug("Dispatch notification record", "record_id", record.ID, "template", templateOutput)

	for _, channel := range channels {
		if err := t.notifierService.Notify(types.NotifyInput{
			Type:     channel.ChannelType,
			To:       channel.ChannelValue,
			Template: templateOutput,
		}); err != nil {
			// todo: The best practice is to record the failure of a channel to avoid repeated pushes, but there will not be multiple channels for the time being
			slog.Warn(
				"Failed to notify",
				"channel_type", channel.ChannelType,
				"channel_to", channel.ChannelValue,
				"error", err,
			)
			return err
		}
	}
	return nil
}
