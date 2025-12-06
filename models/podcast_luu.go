package models

import (
	"time"

	"github.com/google/uuid"
)

type SavedPodcast struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_user_podcast_saved" json:"user_id"`
	PodcastID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_user_podcast_saved" json:"podcast_id"`
	SavedAt   time.Time `gorm:"autoCreateTime" json:"saved_at"`

	User    NguoiDung `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Podcast Podcast   `gorm:"foreignKey:PodcastID;constraint:OnDelete:CASCADE" json:"podcast"`
}

func (SavedPodcast) TableName() string {
	return "saved_podcasts"
}
