package dbmodels

import (
	"time"
)

type Dao struct {
	ID             string     `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	ChainID        int        `gorm:"column:chain_id;not null" json:"chain_id"`
	ChainName      string     `gorm:"column:chain_name;type:varchar(255);not null" json:"chain_name"`
	Name           string     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Code           string     `gorm:"column:code;type:varchar(255);not null;uniqueIndex:uq_dgv_dao_code" json:"code"`
	Seq            int        `gorm:"column:seq;not null;default:0" json:"seq"`
	State          string     `gorm:"column:state;type:varchar(50);not null" json:"state"`
	Tags           string     `gorm:"column:tags;type:text" json:"tags,omitempty"` // Optional tags field
	ConfigLink     string     `gorm:"column:config_link;type:varchar(255);not null" json:"config_link"`
	TimeSyncd      *time.Time `gorm:"column:time_syncd" json:"time_syncd,omitempty"`
	CountProposals int        `gorm:"column:count_proposals;not null;default:0" json:"count_proposals"`
	CTime          time.Time  `gorm:"column:ctime;default:now()" json:"ctime"`
	UTime          *time.Time `gorm:"column:utime" json:"utime,omitempty"`
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
	ChipCode   string     `gorm:"column:chip_code;type:varchar(255);not null" json:"chip_code"`
	Value      string     `gorm:"column:value;type:text;not null" json:"value"`
	Additional string     `gorm:"column:additional;type:text" json:"additional,omitempty"`
	CTime      time.Time  `gorm:"column:ctime;default:now()" json:"ctime"`
	UTime      *time.Time `gorm:"column:utime" json:"utime,omitempty"`
}

func (DgvDaoChip) TableName() string {
	return "dgv_dao_chip"
}
