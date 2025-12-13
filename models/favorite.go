package models

import "time"

type PodcastYeuThich struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	NguoiDungID string    `gorm:"type:char(36);not null;index:idx_fav_user_podcast" json:"nguoi_dung_id"`
	PodcastID   string    `gorm:"type:char(36);not null;index:idx_fav_user_podcast" json:"podcast_id"`
	NgayThich   time.Time `gorm:"autoCreateTime" json:"ngay_thich"`

	NguoiDung NguoiDung `gorm:"foreignKey:NguoiDungID;constraint:OnDelete:CASCADE" json:"nguoi_dung,omitempty"`
	Podcast   Podcast   `gorm:"foreignKey:PodcastID;constraint:OnDelete:CASCADE" json:"podcast,omitempty"`
}
