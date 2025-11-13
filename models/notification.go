package models

import "time"

type Notification struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	UserID    string    `json:"user_id"`              // user nhận notification
	PodcastID string    `json:"podcast_id,omitempty"` // optional
	Action    string    `json:"action"`               // loại action: create_podcast, favorite, ...
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}
