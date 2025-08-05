package dbmodels

import "time"

type ProposalTracking struct {
	ID           string     `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	DaoCode      string     `gorm:"column:dao_code;type:varchar(255);not null" json:"dao_code"`
	ProposalLink string     `gorm:"column:proposal_link;type:varchar(255);not null" json:"proposal_link"`
	ProposalId   string     `gorm:"column:proposal_id;type:varchar(50);not null" json:"proposal_id"`
	State        string     `gorm:"column:state;type:varchar(50);not null" json:"state"`
	CTime        time.Time  `gorm:"column:ctime;default:now()" json:"ctime"`
	UTime        *time.Time `gorm:"column:utime" json:"utime,omitempty"`
}

func (ProposalTracking) TableName() string {
	return "dgv_proposal_tracking"
}
