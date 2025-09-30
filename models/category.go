package models

import "time"

type DanhMuc struct {
	ID         string    `gorm:"type:char(36);primaryKey;default:(UUID())" json:"id"`
	TenDanhMuc string    `gorm:"type:varchar(255)" json:"ten_danh_muc"`
	MoTa       string    `gorm:"type:text" json:"mo_ta"`
	Slug       string    `gorm:"type:varchar(100);uniqueIndex" json:"slug"`
	NgayTao    time.Time `gorm:"autoCreateTime" json:"ngay_tao"`
	KichHoat   bool      `gorm:"default:true" json:"kich_hoat"`
}
