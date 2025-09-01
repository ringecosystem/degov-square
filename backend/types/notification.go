package types

import dbmodels "github.com/ringecosystem/degov-apps/database/models"


type InspectNotificationEventInput struct {
	DaoCode    string
	ProposalID string
	VoteID     *string
	Type       dbmodels.NotificationType
	States     *[]dbmodels.NotificationEventState
}
