package models

import "time"

type PodcastLuu struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	NguoiDungID string    `gorm:"type:char(36);not null" json:"nguoi_dung_id"`
	PodcastID   string    `gorm:"type:char(36);not null" json:"podcast_id"`
	NgayLuu     time.Time `gorm:"autoCreateTime" json:"ngay_luu"`

	// Quan hệ (optional, không bắt buộc nhưng hữu ích khi join)
	NguoiDung NguoiDung `gorm:"foreignKey:NguoiDungID" json:"nguoi_dung,omitempty"`
	Podcast   Podcast   `gorm:"foreignKey:PodcastID" json:"podcast,omitempty"`
}
