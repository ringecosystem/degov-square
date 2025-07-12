package models

import (
	"time"
)

type Dao struct {
	ID         string     `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	ChainID    int        `gorm:"column:chain_id;not null" json:"chain_id"`
	ChainName  string     `gorm:"column:chain_name;type:varchar(255);not null" json:"chain_name"`
	Name       string     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Code       string     `gorm:"column:code;type:varchar(255);not null;uniqueIndex:uq_dgv_dao_code" json:"code"`
	ConfigLink string     `gorm:"column:config_link;type:varchar(255);not null" json:"config_link"`
	TimeSync   *time.Time `gorm:"column:time_sync" json:"time_sync,omitempty"`
	CTime      time.Time  `gorm:"column:ctime;default:now()" json:"ctime"`
}

func (Dao) TableName() string {
	return "dgv_dao"
}
