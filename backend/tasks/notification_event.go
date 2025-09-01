package tasks

import "github.com/ringecosystem/degov-apps/services"

type NotificationEventTask struct {
	daoService     *services.DaoService
	daoChipService *services.DaoChipService
}

func NewNotificationEventTask() *NotificationEventTask {
	return &NotificationEventTask{
		daoService:     services.NewDaoService(),
		daoChipService: services.NewDaoChipService(),
	}
}

func (t *NotificationEventTask) Name() string {
	return "notification-event"
}

func (t *NotificationEventTask) Execute() error {
	return t.buildNotificationRecord()
}

func (t *NotificationEventTask) buildNotificationRecord() error {
	return nil
}
