package dbmodels

import (
	"time"
)

type NotificationType string
type NotificationStatus string

const (
	NotificationTypeNewProposal     NotificationType = "NEW_PROPOSAL"
	NotificationTypeVote            NotificationType = "VOTE"
	NotificationTypeStatus          NotificationType = "STATUS"
	NotificationTypeVoteEndReminder NotificationType = "VOTE_END_REMINDER"
)

const (
	NotificationStatusSentOk   NotificationStatus = "SENT_OK"
	NotificationStatusSentFail NotificationStatus = "SENT_FAIL"
)

type NotificationRecord struct {
	ID         string             `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	ChainID    int                `gorm:"column:chain_id;not null" json:"chain_id"`
	DaoCode    string             `gorm:"column:dao_code;type:varchar(50);not null" json:"dao_code"`
	Type       NotificationType   `gorm:"column:type;type:varchar(50);not null" json:"type"`
	ProposalID string             `gorm:"column:proposal_id;type:varchar(255)" json:"proposal_id"`
	VoteID     *string            `gorm:"column:vote_id;type:varchar(255)" json:"vote_id,omitempty"`
	UserID     string             `gorm:"column:user_id;type:varchar(50);not null" json:"user_id"`
	Status     NotificationStatus `gorm:"column:status;type:varchar(50);not null" json:"status"`
	Message    *string            `gorm:"column:message;type:text" json:"message,omitempty"`
	RetryTimes int                `gorm:"column:retry_times;not null;default:0" json:"retry_times"`
	CTime      time.Time          `gorm:"column:ctime;default:now()" json:"ctime"`
}

func (NotificationRecord) TableName() string {
	return "dgv_notification_record"
}

const (
	ChannelTypeEmail = "EMAIL"
	ChannelTypeSMS   = "SMS"
	ChannelTypePush  = "PUSH"
)

type UserChannel struct {
	ID           string    `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	UserID       string    `gorm:"column:user_id;type:varchar(50);not null" json:"user_id"`
	Verified     int       `gorm:"column:verified;not null;default:0" json:"verified"`
	ChannelType  string    `gorm:"column:channel_type;type:varchar(50);not null" json:"channel_type"`
	ChannelValue string    `gorm:"column:channel_value;type:varchar(500);not null" json:"channel_value"`
	Payload      *string   `gorm:"column:payload;type:text" json:"payload,omitempty"`
	CTime        time.Time `gorm:"column:ctime;default:now()" json:"ctime"`
}

func (UserChannel) TableName() string {
	return "dgv_user_channel"
}
