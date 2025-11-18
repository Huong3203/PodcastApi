package models

import (
	"time"
)

type Payment struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	OrderID   string    `gorm:"type:varchar(50);index"   json:"order_id"`
	UserID    string    `gorm:"type:char(36);not null"   json:"user_id"`
	Amount    int       `gorm:"type:int;not null"        json:"amount"`
	Status    string    `gorm:"type:varchar(20);default:'pending'" json:"status"`
	PodcastID *string   `gorm:"type:char(36);default:NULL" json:"podcast_id"` // VIP payment để nil
	CreatedAt time.Time `gorm:"autoCreateTime"           json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"           json:"updated_at"`
}
