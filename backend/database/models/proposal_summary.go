package dbmodels

import "time"

// ProposalSummary represents an AI-generated summary of a governance proposal
type ProposalSummary struct {
	ID          string     `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	DaoCode     *string    `gorm:"column:dao_code;type:varchar(255)" json:"dao_code,omitempty"`
	ChainId     int        `gorm:"column:chain_id;not null" json:"chain_id"`
	ProposalID  string     `gorm:"column:proposal_id;type:varchar(255);not null" json:"proposal_id"`
	Indexer     *string    `gorm:"column:indexer;type:varchar(255)" json:"indexer,omitempty"`
	Description string     `gorm:"column:description;type:text;not null" json:"description"`
	Summary     string     `gorm:"column:summary;type:text;not null" json:"summary"`
	CTime       time.Time  `gorm:"column:ctime;default:now()" json:"ctime"`
	UTime       *time.Time `gorm:"column:utime" json:"utime,omitempty"`
}

func (ProposalSummary) TableName() string {
	return "dgv_proposal_summary"
}
