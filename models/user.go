package models

import "time"

type NguoiDung struct {
	ID       string    `gorm:"type:char(36);primaryKey" json:"id"`
	Email    string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	MatKhau  string    `gorm:"type:varchar(255);not null" json:"-"`
	HoTen    string    `gorm:"type:varchar(100)" json:"ho_ten"`
	VaiTro   string    `gorm:"type:varchar(20);default:'user';check:vai_tro IN ('admin','user')" json:"vai_tro"`
	Avatar   string    `gorm:"type:varchar(255);default:''" json:"avatar"`
	NgayTao  time.Time `gorm:"autoCreateTime" json:"ngay_tao"`
	KichHoat bool      `gorm:"default:true" json:"kich_hoat"`
}
