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
	NoiDungTrichXuat string     `gorm:"type:longtext" json:"noi_dung_trich_xuat"`
	TrangThai        string     `gorm:"type:enum('Đã tải lên', 'Đã kiểm tra', 'Đã trích xuất', 'Đã xử lý AI', 'Hoàn thành', 'Đã xuất bản')" json:"trang_thai"`
	NguoiTaiLen      string     `gorm:"type:char(36);not null" json:"nguoi_tai_len"`
	NgayTaiLen       time.Time  `gorm:"autoCreateTime" json:"ngay_tai_len"`
	NgayXuLyXong     *time.Time `gorm:"" json:"ngay_xu_ly_xong"`
	// Định nghĩa khóa ngoại
	NguoiDung NguoiDung `gorm:"foreignKey:NguoiTaiLen;references:ID" json:"nguoi_dung"`
}
