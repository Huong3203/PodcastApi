package models

import (
	"time"
)

// ==================== LỊCH SỬ NGHE ====================
type LichSuNghe struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	NguoiDungID string    `gorm:"type:char(36);not null;index:idx_user_podcast" json:"nguoi_dung_id"`
	PodcastID   string    `gorm:"type:char(36);not null;index:idx_user_podcast" json:"podcast_id"`
	ViTri       int       `gorm:"default:0" json:"vi_tri"`         // Vị trí nghe (giây)
	NgayNghe    time.Time `gorm:"autoUpdateTime" json:"ngay_nghe"` // Lần nghe gần nhất

	NguoiDung NguoiDung `gorm:"foreignKey:NguoiDungID;constraint:OnDelete:CASCADE" json:"nguoi_dung,omitempty"`
	Podcast   Podcast   `gorm:"foreignKey:PodcastID;constraint:OnDelete:CASCADE" json:"podcast,omitempty"`
}
