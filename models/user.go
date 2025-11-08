package models

import (
	"time"
)

type NguoiDung struct {
	ID       string    `gorm:"type:char(36);primaryKey" json:"id"`
	Email    string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	MatKhau  string    `gorm:"type:varchar(255);not null" json:"-"` // ẩn mật khẩu khi trả về
	HoTen    string    `gorm:"type:varchar(100)" json:"ho_ten"`
	VaiTro   string    `gorm:"type:enum('admin', 'user');default:'user'" json:"vai_tro"`
	Avatar   string    `gorm:"type:varchar(255);default:''" json:"avatar"`
	NgayTao  time.Time `gorm:"autoCreateTime" json:"ngay_tao"`
	KichHoat bool      `gorm:"default:true" json:"kich_hoat"`
	Provider string    `gorm:"type:enum('local', 'clerk');default:'local'" json:"provider"`
}
