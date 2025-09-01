package tasks

import "github.com/ringecosystem/degov-apps/services"

type NotificationDispatcherTask struct {
	daoService     *services.DaoService
	daoChipService *services.DaoChipService
}

func NewNotificationDispatcherTask() *NotificationDispatcherTask {
	return &NotificationDispatcherTask{
		daoService:     services.NewDaoService(),
		daoChipService: services.NewDaoChipService(),
	}
}

func (t *NotificationDispatcherTask) Name() string {
	return "notification-dispatcher"
}

func (t *NotificationDispatcherTask) Execute() error {
	return t.DispatcherNotificationRecord()
}

func (t *NotificationDispatcherTask) DispatcherNotificationRecord() error {
	return nil
}
