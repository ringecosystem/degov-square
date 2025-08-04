package types

import "time"

type AuthenticatedUserKeyType struct{}

type UserSessInfo struct {
	Id      string     `json:"id"`
	Address string     `json:"address"`
	Email   *string    `json:"email,omitempty"`
	CTime   time.Time  `json:"ctime"`
	UTime   *time.Time `json:"utime"`
}
