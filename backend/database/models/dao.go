package dbmodels

import (
	"time"
)

type ChipCode string

const (
	ChipCodeAgent        ChipCode = "AGENT"
	ChipCodeMetricsState ChipCode = "METRICS_STATE"
)

type Dao struct {
	ID                    string     `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	ChainID               int        `gorm:"column:chain_id;not null" json:"chain_id"`
	ChainName             string     `gorm:"column:chain_name;type:varchar(255);not null" json:"chain_name"`
	Name                  string     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Code                  string     `gorm:"column:code;type:varchar(255);not null;uniqueIndex:uq_dgv_dao_code" json:"code"`
	Seq                   int        `gorm:"column:seq;not null;default:0" json:"seq"`
	Endpoint              string     `gorm:"column:endpoint;type:varchar(255);not null" json:"endpoint"` // Website endpoint
	State                 string     `gorm:"column:state;type:varchar(50);not null" json:"state"`
	Tags                  string     `gorm:"column:tags;type:text" json:"tags,omitempty"` // Optional tags field
	ConfigLink            string     `gorm:"column:config_link;type:varchar(255);not null" json:"config_link"`
	TimeSyncd             *time.Time `gorm:"column:time_syncd" json:"time_syncd,omitempty"`
	MetricsCountProposals int        `gorm:"column:metrics_count_proposals;not null;default:0" json:"metrics_count_proposals"`
	MetricsCountMembers   int        `gorm:"column:metrics_count_members;not null;default:0" json:"metrics_count_members"`
	MetricsSumPower       string     `gorm:"column:metrics_sum_power;type:varchar(255);not null;default:'0'" json:"metrics_sum_power"`
	MetricsCountVote      int        `gorm:"column:metrics_count_vote;not null;default:0" json:"metrics_count_vote"`
	LastTrackingBlock     int        `gorm:"column:last_tracking_block;not null;default:0" json:"last_tracking_block"` // Last block number tracked for this DAO
	CTime                 time.Time  `gorm:"column:ctime;default:now()" json:"ctime"`
	UTime                 *time.Time `gorm:"column:utime" json:"utime,omitempty"`
}

func (Dao) TableName() string {
	return "dgv_dao"
}

type DgvDaoConfig struct {
	ID      string     `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	DaoCode string     `gorm:"column:dao_code;type:varchar(255);not null;uniqueIndex:uq_dgv_dao_config_code" json:"dao_code"`
	Config  string     `gorm:"column:config;type:text;not null" json:"config"`
	CTime   time.Time  `gorm:"column:ctime;default:now()" json:"ctime"`
	UTime   *time.Time `gorm:"column:utime" json:"utime,omitempty"`
}

func (DgvDaoConfig) TableName() string {
	return "dgv_dao_config"
}

type DgvDaoChip struct {
	ID         string     `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	DaoCode    string     `gorm:"column:dao_code;type:varchar(255);not null;uniqueIndex:uq_dgv_dao_chip_code" json:"dao_code"`
	ChipCode   ChipCode   `gorm:"column:chip_code;type:varchar(255);not null" json:"chip_code"`
	Flag       string     `gorm:"column:flag;type:varchar(255)" json:"flag,omitempty"` // Optional flag for chip
	Additional string     `gorm:"column:additional;type:text" json:"additional,omitempty"`
	CTime      time.Time  `gorm:"column:ctime;default:now()" json:"ctime"`
	UTime      *time.Time `gorm:"column:utime" json:"utime,omitempty"`
}

func (DgvDaoChip) TableName() string {
	return "dgv_dao_chip"
}
