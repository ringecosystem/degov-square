package tasks

import "github.com/ringecosystem/degov-apps/services"

type NotificationPushTask struct {
	daoService     *services.DaoService
	daoChipService *services.DaoChipService
}

func NewNotificationPushTask() *NotificationPushTask {
	return &NotificationPushTask{
		daoService:     services.NewDaoService(),
		daoChipService: services.NewDaoChipService(),
	}
}

func (t *NotificationPushTask) Name() string {
	return "notification-push"
}

func (t *NotificationPushTask) Execute() error {
	return t.pushNotificationRecord()
}

func (t *NotificationPushTask) pushNotificationRecord() error {
	return nil
}
