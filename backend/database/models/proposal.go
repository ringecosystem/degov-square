package dbmodels

import "time"

type ProposalState string

const (
	ProposalStateUnknown   ProposalState = "UNKNOWN"
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
	ID                 string        `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	DaoCode            string        `gorm:"column:dao_code;type:varchar(255);not null" json:"dao_code"`
	ChainId            int           `gorm:"column:chain_id;not null" json:"chain_id"` // Chain ID for the DAO
	Title              string        `gorm:"column:title;type:varchar(500);not null" json:"title"`
	ProposalLink       string        `gorm:"column:proposal_link;type:varchar(255);not null" json:"proposal_link"`
	ProposalID         string        `gorm:"column:proposal_id;type:varchar(50);not null" json:"proposal_id"`
	State              ProposalState `gorm:"column:state;type:varchar(50);not null" json:"state"`
	ProposalCreatedAt  *time.Time    `gorm:"column:proposal_created_at" json:"proposal_created_at,omitempty"` // Proposal creation time
	ProposalAtBlock    int           `gorm:"column:proposal_at_block;not null" json:"proposal_at_block"`      // Block number when the proposal was created
	TimesTrack         int           `gorm:"column:times_track;not null;default:0" json:"times_track"`        // Number of times the proposal has been tracked
	TimeNextTrack      *time.Time    `gorm:"column:time_next_track" json:"time_next_track,omitempty"`         // Next tracking time
	Message            string        `gorm:"column:message;type:text" json:"message,omitempty"`               // Additional message or notes
	OffsetTrackingVote int           `gorm:"column:offset_tracking_vote;default:0" json:"offset_tracking_vote"`

	// Fulfill fields for AI agent voting
	Fulfilled        int        `gorm:"column:fulfilled;default:0" json:"fulfilled"`                 // 0: not fulfilled, 1: fulfilled
	FulfilledExplain *string    `gorm:"column:fulfilled_explain;type:text" json:"fulfilled_explain"` // AI decision explanation
	FulfilledAt      *time.Time `gorm:"column:fulfilled_at" json:"fulfilled_at,omitempty"`           // Time when fulfilled
	TimesFulfill     int        `gorm:"column:times_fulfill;default:0" json:"times_fulfill"`         // Number of fulfill attempts
	FulfillErrored   int        `gorm:"column:fulfill_errored;default:0" json:"fulfill_errored"`     // 0: no error, 1: errored after max retries

	CTime time.Time  `gorm:"column:ctime;default:now()" json:"ctime"`
	UTime *time.Time `gorm:"column:utime" json:"utime,omitempty"`
}

func (ProposalTracking) TableName() string {
	return "dgv_proposal_tracking"
}
