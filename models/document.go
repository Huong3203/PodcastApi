package models

import (
	"time"
)

type TaiLieu struct {
	ID               string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenFileGoc       string     `gorm:"type:varchar(255)" json:"ten_file_goc"`
	DuongDanFile     string     `gorm:"type:text" json:"duong_dan_file"`
	LoaiFile         string     `gorm:"type:varchar(50)" json:"loai_file"`
	KichThuocFile    int64      `gorm:"type:int" json:"kich_thuoc_file"`
	NoiDungTrichXuat string     `gorm:"type:text" json:"noi_dung_trich_xuat"`
	TrangThai        string     `gorm:"type:varchar(50);check:trang_thai IN ('Đã tải lên','Đã kiểm tra','Đã trích xuất','Đã xử lý AI','Hoàn thành','Đã xuất bản')" json:"trang_thai"` // ✅ enum -> check
	NguoiTaiLen      string     `gorm:"type:char(36);not null" json:"nguoi_tai_len"`
	NgayTaiLen       time.Time  `gorm:"autoCreateTime" json:"ngay_tai_len"`
	NgayXuLyXong     *time.Time `json:"ngay_xu_ly_xong"`

	NguoiDung NguoiDung `gorm:"foreignKey:NguoiTaiLen;references:ID" json:"nguoi_dung"`
}
