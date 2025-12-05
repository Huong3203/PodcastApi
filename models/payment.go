package models

import "time"

type Payment struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	UserID      string    `gorm:"type:char(36);not null;index" json:"user_id"`
	OrderID     string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"order_id"`
	RequestID   string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"request_id"`
	Amount      int64     `gorm:"not null" json:"amount"`
	Status      string    `gorm:"type:enum('PENDING','SUCCESS','FAILED','CANCELLED');default:'PENDING'" json:"status"`
	PaymentType string    `gorm:"type:varchar(50);default:'VIP_UPGRADE'" json:"payment_type"`
	VIPDuration int       `gorm:"default:30" json:"vip_duration"`
	PaymentInfo string    `gorm:"type:text" json:"payment_info"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
