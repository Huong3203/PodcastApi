package models

import (
	"time"

	"github.com/google/uuid"
)

type ListeningHistory struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_user_podcast" json:"user_id"`
	PodcastID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_user_podcast" json:"podcast_id"`

	ListenedAt time.Time `gorm:"autoUpdateTime" json:"listened_at"` // Lần nghe gần nhất

	User    NguoiDung `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Podcast Podcast   `gorm:"foreignKey:PodcastID;constraint:OnDelete:CASCADE" json:"podcast"`
}

func (ListeningHistory) TableName() string {
	return "listening_histories"
}
