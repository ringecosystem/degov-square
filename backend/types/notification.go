package types

import dbmodels "github.com/ringecosystem/degov-apps/database/models"

type InspectNotificationEventInput struct {
	DaoCode    string
	ProposalID string
	VoteID     *string
	Type       dbmodels.SubscribeFeatureName
	States     *[]dbmodels.NotificationEventState
}

type ListLimitEventsInput struct {
	Limit  int
	States *[]dbmodels.NotificationEventState
}

type UpdateEventStateInput struct {
	ID    string
	State dbmodels.NotificationEventState
}

type UpdateEventRetryTimes struct {
	ID         string
	TimesRetry int
}
