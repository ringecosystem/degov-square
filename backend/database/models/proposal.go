package dbmodels

import "time"

type ProposalState string

const (
	ProposalStatePending   ProposalState = "PENDING"
	ProposalStateActive    ProposalState = "ACTIVE"
	ProposalStateCanceled  ProposalState = "CANCELED"
	ProposalStateDefeated  ProposalState = "DEFEATED"
	ProposalStateSucceeded ProposalState = "SUCCEEDED"
	ProposalStateQueued    ProposalState = "QUEUED"
	ProposalStateExecuted  ProposalState = "EXECUTED"
	ProposalStateExpired   ProposalState = "EXPIRED"
)

type ProposalTracking struct {
	ID                string        `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	DaoCode           string        `gorm:"column:dao_code;type:varchar(255);not null" json:"dao_code"`
	ProposalLink      string        `gorm:"column:proposal_link;type:varchar(255);not null" json:"proposal_link"`
	ProposalId        string        `gorm:"column:proposal_id;type:varchar(50);not null" json:"proposal_id"`
	State             ProposalState `gorm:"column:state;type:varchar(50);not null" json:"state"`
	ProposalCreatedAt *time.Time    `gorm:"column:proposal_created_at" json:"proposal_created_at,omitempty"` // Proposal creation time
	ProposalAtBlock   int           `gorm:"column:proposal_at_block;not null" json:"proposal_at_block"`      // Block number when the proposal was created
	CTime             time.Time     `gorm:"column:ctime;default:now()" json:"ctime"`
	UTime             *time.Time    `gorm:"column:utime" json:"utime,omitempty"`
}

func (ProposalTracking) TableName() string {
	return "dgv_proposal_tracking"
}
