package models

import "time"

type LichSuNghe struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	NguoiDungID string    `gorm:"type:char(36);not null" json:"nguoi_dung_id"`
	PodcastID   string    `gorm:"type:char(36);not null" json:"podcast_id"`
	ViTri       int       `gorm:"type:int;default:0" json:"vi_tri"`
	NgayNghe    time.Time `gorm:"autoCreateTime" json:"ngay_nghe"`

	NguoiDung NguoiDung `gorm:"foreignKey:NguoiDungID;references:ID" json:"nguoi_dung,omitempty"`
	Podcast   Podcast   `gorm:"foreignKey:PodcastID;references:ID" json:"podcast,omitempty"`
}
