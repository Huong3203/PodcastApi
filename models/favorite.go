package models

import (
	"time"

	"github.com/google/uuid"
)

type Favorite struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_user_podcast_favorite" json:"user_id"`
	PodcastID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_user_podcast_favorite" json:"podcast_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	User    NguoiDung `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Podcast Podcast   `gorm:"foreignKey:PodcastID;constraint:OnDelete:CASCADE" json:"podcast"`
}

func (Favorite) TableName() string {
	return "favorites"
}
