package tasks

import (
	"log/slog"

	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	"github.com/ringecosystem/degov-apps/services"
	"github.com/ringecosystem/degov-apps/types"
)

type NotificationDispatcherTask struct {
	notificationService *services.NotificationService
	templateService     *services.TemplateService
}

func NewNotificationDispatcherTask() *NotificationDispatcherTask {
	return &NotificationDispatcherTask{
		notificationService: services.NewNotificationService(),
		templateService:     services.NewTemplateService(),
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
		if err := t.dispatchNotificationRecordByRecord(&record); err != nil {
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

func (t *NotificationDispatcherTask) dispatchNotificationRecordByRecord(record *dbmodels.NotificationRecord) error {
	template, err := t.templateService.GenerateTemplateByNotificationRecord(record)
	if err != nil {
		return err
	}
	// return t.notificationService.SendNotification(record, template)
	slog.Debug("Dispatch notification record", "record_id", record.ID, "template", template)
	return nil
}
