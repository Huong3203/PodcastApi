package models

import "time"

type Notification struct {
	ID        string     `gorm:"type:char(36);primaryKey;default:(UUID())" json:"id"`
	UserID    string     `gorm:"type:char(36);not null" json:"user_id"`
	PodcastID string     `gorm:"type:char(36)" json:"podcast_id,omitempty"`
	Action    string     `gorm:"type:varchar(50)" json:"action"`
	Message   string     `gorm:"type:text" json:"message"`
	IsRead    bool       `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}
