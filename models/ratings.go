package models

import (
	"time"
)

type DanhGia struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	PodcastID string    `gorm:"type:char(36);not null" json:"podcast_id"`
	UserID    string    `gorm:"type:char(36);not null" json:"user_id"`
	Sao       int       `gorm:"type:int;not null" json:"sao"` // 1-5 sao
	BinhLuan  string    `gorm:"type:text" json:"binh_luan"`
	NgayTao   time.Time `gorm:"autoCreateTime" json:"ngay_tao"`

	// Khóa ngoại
	Podcast Podcast   `gorm:"foreignKey:PodcastID;references:ID" json:"podcast"`
	User    NguoiDung `gorm:"foreignKey:UserID;references:ID" json:"user"`
}
