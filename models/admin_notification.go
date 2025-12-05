// models/admin_notification.go
package models

import (
	"time"

	"github.com/google/uuid"
)

type AdminNotification struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	AdminID   uuid.UUID  `gorm:"type:uuid;not null" json:"admin_id"`
	Title     string     `gorm:"size:255;not null" json:"title"`
	Message   string     `gorm:"type:text;not null" json:"message"`
	Type      string     `gorm:"size:50" json:"type"` // new_podcast, new_user, report, etc.
	IsRead    bool       `gorm:"default:false" json:"is_read"`
	PodcastID *uuid.UUID `gorm:"type:uuid" json:"podcast_id,omitempty"`
	UserID    *uuid.UUID `gorm:"type:uuid" json:"user_id,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}
