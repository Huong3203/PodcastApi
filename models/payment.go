package models

import (
	"time"
)

type Payment struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    string    `gorm:"type:char(36);not null" json:"user_id"` // user mua gói VIP
	Amount    int       `gorm:"type:int;not null" json:"amount"`
	Status    string    `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending, success, failed
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
