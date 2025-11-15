package models

import "time"

type FeaturedRating struct {
	ID         string    `gorm:"type:char(36);primaryKey" json:"id"`
	RatingID   string    `gorm:"type:char(36);not null" json:"rating_id"`
	PodcastID  string    `gorm:"type:char(36);not null" json:"podcast_id"`
	UserID     string    `gorm:"type:char(36);not null" json:"user_id"`
	Sao        int       `json:"sao"`
	BinhLuan   string    `json:"binh_luan"`
	FeaturedAt time.Time `json:"featured_at"`

	User    NguoiDung `gorm:"foreignKey:UserID;references:ID" json:"user"`
	Podcast Podcast   `gorm:"foreignKey:PodcastID;references:ID" json:"podcast"`
}

func (FeaturedRating) TableName() string {
	return "featured_rating" // 👈 bảng thật
}
