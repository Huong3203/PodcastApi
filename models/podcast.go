package models

import (
	"time"
)

type Podcast struct {
	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
	TailieuID      string     `gorm:"type:char(36);not null" json:"tai_lieu_id"`
	TieuDe         string     `gorm:"type:varchar(255)" json:"tieu_de"`
	MoTa           string     `gorm:"type:text" json:"mo_ta"`
	DuongDanAudio  string     `gorm:"type:text" json:"duong_dan_audio"`
	ThoiLuongGiay  int        `gorm:"type:int" json:"thoi_luong_giay"`
	HinhAnhDaiDien string     `gorm:"type:text" json:"hinh_anh_dai_dien"`
	DanhMucID      string     `gorm:"type:char(36);not null" json:"danh_muc_id"`
	TrangThai      string     `gorm:"type:enum('Tắt','Bật'); default:'Tắt'" json:"trang_thai"`
	NguoiTao       string     `gorm:"type:char(36);not null" json:"nguoi_tao"`
	NgayTaoRa      time.Time  `gorm:"autoCreateTime" json:"ngay_tao_ra"`
	NgayXuatBan    *time.Time `gorm:"" json:"ngay_xuat_ban"`
	TheTag         string     `gorm:"type:varchar(255)" json:"the_tag"`
	LuotXem        int        `gorm:"type:int;default:0" json:"luot_xem"`
	// Định nghĩa khóa ngoại
	TaiLieu TaiLieu `gorm:"foreignKey:TailieuID;references:ID" json:"tailieu"`
	DanhMuc DanhMuc `gorm:"foreignKey:DanhMucID;references:ID" json:"danhmuc"`
}
