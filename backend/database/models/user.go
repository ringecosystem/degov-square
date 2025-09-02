package dbmodels

import "time"

type User struct {
	ID      string     `gorm:"column:id;type:varchar(50);primaryKey" json:"id"`
	Address string     `gorm:"column:address;type:varchar(255);not null;uniqueIndex:uq_dgv_user_address" json:"address"`
	Email   *string    `gorm:"column:email;type:varchar(255)" json:"email,omitempty"`
	EnsName *string    `gorm:"column:ens_name;type:varchar(255)" json:"ens_name,omitempty"`
	CTime   time.Time  `gorm:"column:ctime;default:now()" json:"ctime"`
	UTime   *time.Time `gorm:"column:utime" json:"utime,omitempty"`
}

func (User) TableName() string {
	return "dgv_user"
}
