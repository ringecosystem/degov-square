package dbmodels

import "time"

type ProposalCommentState string

const (
	ProposalCommentStateActive  ProposalCommentState = "ACTIVE"
	ProposalCommentStateDeleted ProposalCommentState = "DELETED"
)

type ProposalComment struct {
	ID          string               `gorm:"column:id;type:varchar(50);primaryKey"`
	DaoCode     string               `gorm:"column:dao_code;type:varchar(255);not null"`
	ChainID     int                  `gorm:"column:chain_id;not null"`
	ProposalID  string               `gorm:"column:proposal_id;type:varchar(78);not null"`
	UserID      string               `gorm:"column:user_id;type:varchar(50);not null"`
	UserAddress string               `gorm:"column:user_address;type:varchar(255);not null"`
	ReplyToID   *string              `gorm:"column:reply_to_id;type:varchar(50)"`
	Body        string               `gorm:"column:body;type:text;not null"`
	State       ProposalCommentState `gorm:"column:state;type:varchar(20);not null"`
	CTime       time.Time            `gorm:"column:ctime;not null;default:now()"`
	UTime       *time.Time           `gorm:"column:utime"`
}

func (ProposalComment) TableName() string {
	return "dgv_proposal_comment"
}
