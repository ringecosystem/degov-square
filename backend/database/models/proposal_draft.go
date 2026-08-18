package dbmodels

import "time"

type ProposalDraft struct {
	ID              string    `gorm:"column:id;type:varchar(50);primaryKey"`
	ClientRequestID string    `gorm:"column:client_request_id;type:varchar(100);not null"`
	DaoCode         string    `gorm:"column:dao_code;type:varchar(255);not null"`
	ChainID         int       `gorm:"column:chain_id;not null"`
	UserID          string    `gorm:"column:user_id;type:varchar(50);not null"`
	UserAddress     string    `gorm:"column:user_address;type:varchar(255);not null"`
	Title           string    `gorm:"column:title;type:varchar(200);not null"`
	Payload         string    `gorm:"column:payload;type:jsonb;not null"`
	PayloadVersion  int       `gorm:"column:payload_version;not null"`
	Revision        int       `gorm:"column:revision;not null"`
	CTime           time.Time `gorm:"column:ctime;not null;default:now()"`
	UTime           time.Time `gorm:"column:utime;not null;default:now()"`
}

func (ProposalDraft) TableName() string {
	return "dgv_proposal_draft"
}
