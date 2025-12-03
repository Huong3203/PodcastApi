package models

import (
	"time"
)

type NguoiDung struct {
	ID         string     `gorm:"type:char(36);primaryKey" json:"id"`
	Email      string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	MatKhau    string     `gorm:"type:varchar(255)" json:"-"`
	HoTen      string     `gorm:"type:varchar(100)" json:"ho_ten"`
	VaiTro     string     `gorm:"type:enum('admin', 'user');default:'user'" json:"vai_tro"`
	Avatar     string     `gorm:"type:varchar(255);default:''" json:"avatar"`
	NgayTao    time.Time  `gorm:"autoCreateTime" json:"ngay_tao"`
	KichHoat   bool       `gorm:"default:true" json:"kich_hoat"`
	Provider   string     `gorm:"type:enum('local', 'clerk');default:'local'" json:"provider"`
	VIP        bool       `gorm:"column:vip;default:false" json:"vip"`
	VIPExpires *time.Time `gorm:"column:vip_expires;type:timestamp;default:NULL" json:"vip_expires"`
	AutoRenew  bool       `gorm:"column:auto_renew;default:false" json:"auto_renew"`
}
