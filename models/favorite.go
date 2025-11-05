package models

import "time"

type PodcastYeuThich struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	NguoiDungID string    `gorm:"type:char(36);not null" json:"nguoi_dung_id"`
	PodcastID   string    `gorm:"type:char(36);not null" json:"podcast_id"`
	NgayThich   time.Time `gorm:"autoCreateTime" json:"ngay_thich"`

	NguoiDung NguoiDung `gorm:"foreignKey:NguoiDungID" json:"nguoi_dung,omitempty"`
	Podcast   Podcast   `gorm:"foreignKey:PodcastID" json:"podcast,omitempty"`
}
