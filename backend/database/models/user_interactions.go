package dbmodels

import (
	"time"
)

type UserLikedDao struct {
	ID          string    `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	DaoCode     string    `gorm:"column:dao_code;type:varchar(50);not null" json:"dao_code"`
	UserID      string    `gorm:"column:user_id;type:varchar(50);not null;uniqueIndex:uq_dgv_user_liked_dao_code_uid,priority:1" json:"user_id"`
	UserAddress string    `gorm:"column:user_address;type:varchar(255);not null;uniqueIndex:uq_dgv_user_liked_dao_code_address,priority:2" json:"user_address"`
	CTime       time.Time `gorm:"column:ctime;default:now()" json:"ctime"`
}

func (UserLikedDao) TableName() string {
	return "dgv_user_liked_dao"
}

type SubscribeFeature string

const (
	SubscribeFeatureEnableProposal          SubscribeFeature = "ENABLE_PROPOSAL"
	SubscribeFeatureEnableVotingEndReminder SubscribeFeature = "ENABLE_VOTING_END_REMINDER"
)

type UserSubscribedDao struct {
	ID          string           `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	ChainID     int              `gorm:"column:chain_id;not null" json:"chain_id"`
	DaoCode     string           `gorm:"column:dao_code;type:varchar(255);not null" json:"dao_code"`
	UserID      string           `gorm:"column:user_id;type:varchar(50);not null;" json:"user_id"`
	UserAddress string           `gorm:"column:user_address;type:varchar(255);not null;" json:"user_address"`
	State       string           `gorm:"column:state;type:varchar(50);not null" json:"state"` // SUBSCRIBED, UNSUBSCRIBED
	Feature     SubscribeFeature `gorm:"column:feature;type:varchar(255);not null" json:"feature"`
	Strategy    string           `gorm:"column:strategy;type:varchar(255);not null" json:"strategy"`
	CTime       time.Time        `gorm:"column:ctime;default:now()" json:"ctime"`
}

func (UserSubscribedDao) TableName() string {
	return "dgv_user_subscribed_dao"
}
