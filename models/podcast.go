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
	NgayXuatBan    *time.Time `json:"ngay_xuat_ban"`
	TheTag         string     `gorm:"type:varchar(255)" json:"the_tag"`
	LuotXem        int        `gorm:"type:int;default:0" json:"luot_xem"`

	// ⭐ Field VIP (đã fix chuẩn MySQL)
	IsVIP bool `gorm:"column:is_vip;type:TINYINT(1);default:0" json:"is_vip"`

	// 2 trường thống kê
	LuotLuu      int `gorm:"type:int;default:0" json:"luot_luu"`
	LuotYeuThich int `gorm:"type:int;default:0" json:"luot_yeu_thich"`

	// Quan hệ
	TaiLieu TaiLieu `gorm:"foreignKey:TailieuID;references:ID" json:"tailieu"`
	DanhMuc DanhMuc `gorm:"foreignKey:DanhMucID;references:ID" json:"danhmuc"`

	// Lấy tóm tắt từ TaiLieu (không lưu vào DB)
	TomTat string `gorm:"-" json:"tom_tat"`
}

// package models

// import (
// 	"time"
// )

// type Podcast struct {
// 	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
// 	TailieuID      string     `gorm:"type:char(36);not null" json:"tai_lieu_id"`
// 	TieuDe         string     `gorm:"type:varchar(255)" json:"tieu_de"`
// 	MoTa           string     `gorm:"type:text" json:"mo_ta"`
// 	DuongDanAudio  string     `gorm:"type:text" json:"duong_dan_audio"`
// 	ThoiLuongGiay  int        `gorm:"type:int" json:"thoi_luong_giay"`
// 	HinhAnhDaiDien string     `gorm:"type:text" json:"hinh_anh_dai_dien"`
// 	DanhMucID      string     `gorm:"type:char(36);not null" json:"danh_muc_id"`
// 	TrangThai      string     `gorm:"type:enum('Tắt','Bật'); default:'Tắt'" json:"trang_thai"`
// 	NguoiTao       string     `gorm:"type:char(36);not null" json:"nguoi_tao"`
// 	NgayTaoRa      time.Time  `gorm:"autoCreateTime" json:"ngay_tao_ra"`
// 	NgayXuatBan    *time.Time `json:"ngay_xuat_ban"`
// 	TheTag         string     `gorm:"type:varchar(255)" json:"the_tag"`
// 	LuotXem        int        `gorm:"type:int;default:0" json:"luot_xem"`
// 	IsVIP          bool       `gorm:"type:boolean;default:false" json:"is_vip"`

// 	// 2 trường thống kê
// 	LuotLuu      int `gorm:"type:int;default:0" json:"luot_luu"`
// 	LuotYeuThich int `gorm:"type:int;default:0" json:"luot_yeu_thich"`

// 	// Quan hệ
// 	TaiLieu TaiLieu `gorm:"foreignKey:TailieuID;references:ID" json:"tailieu"`
// 	DanhMuc DanhMuc `gorm:"foreignKey:DanhMucID;references:ID" json:"danhmuc"`

// 	// Lấy tóm tắt từ TaiLieu để hiển thị trực tiếp trong Podcast
// 	TomTat string `gorm:"-" json:"tom_tat"` // "-" nghĩa là GORM không map vào DB
// }
