package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
)

//
// ================== PUBLIC (Không cần đăng nhập) ==================
//

// ✅ Public: Lấy danh sách danh mục (phân trang, tìm kiếm, lọc trạng thái)
func GetDanhMucs(c *gin.Context) {
	var danhMucs []models.DanhMuc
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	search := c.Query("search")
	status := c.Query("status")

	query := config.DB.Model(&models.DanhMuc{})

	// Lấy vai trò từ middleware (nếu có)
	role, _ := c.Get("vai_tro")

	// ✅ Người dùng thường chỉ thấy danh mục được kích hoạt
	if role != "admin" {
		query = query.Where("kich_hoat = ?", true)
	}

	if search != "" {
		query = query.Where("LOWER(ten_danh_muc) LIKE ?", "%"+strings.ToLower(search)+"%")
	}

	// ✅ Admin mới được lọc theo trạng thái
	if status != "" && role == "admin" {
		switch strings.ToLower(status) {
		case "true":
			query = query.Where("kich_hoat = ?", true)
		case "false":
			query = query.Where("kich_hoat = ?", false)
		}
	}

	query.Count(&total)
	query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&danhMucs)

	c.JSON(http.StatusOK, gin.H{
		"data": danhMucs,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		},
	})
}

//
// ================== ADMIN (Cần đăng nhập + role = admin) ==================
//

// ✅ Tạo danh mục mới
func CreateDanhMuc(c *gin.Context) {
	if role, _ := c.Get("vai_tro"); role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền tạo danh mục"})
		return
	}

	var input struct {
		TenDanhMuc string `json:"ten_danh_muc" binding:"required"`
		MoTa       string `json:"mo_ta"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	slugStr := slug.Make(input.TenDanhMuc)
	danhMuc := models.DanhMuc{
		ID:         uuid.New().String(),
		TenDanhMuc: input.TenDanhMuc,
		Slug:       slugStr,
		MoTa:       input.MoTa,
		KichHoat:   true,
	}

	if err := config.DB.Create(&danhMuc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo danh mục"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tạo danh mục thành công",
		"data":    danhMuc,
	})
}

// ✅ Cập nhật danh mục
func UpdateDanhMuc(c *gin.Context) {
	if role, _ := c.Get("vai_tro"); role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền cập nhật danh mục"})
		return
	}

	var input struct {
		TenDanhMuc string `json:"ten_danh_muc" binding:"required"`
		MoTa       string `json:"mo_ta"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	id := c.Param("id")
	var danhMuc models.DanhMuc
	if err := config.DB.First(&danhMuc, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy danh mục"})
		return
	}

	danhMuc.TenDanhMuc = input.TenDanhMuc
	danhMuc.MoTa = input.MoTa
	danhMuc.Slug = slug.Make(input.TenDanhMuc)
	config.DB.Save(&danhMuc)

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật danh mục thành công",
		"data":    danhMuc,
	})
}

// ✅ Bật / Tắt danh mục
func ToggleDanhMucStatus(c *gin.Context) {
	if role, _ := c.Get("vai_tro"); role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền thay đổi trạng thái danh mục"})
		return
	}

	id := c.Param("id")
	var body struct {
		KichHoat bool `json:"kich_hoat"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	var dm models.DanhMuc
	if err := config.DB.First(&dm, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy danh mục"})
		return
	}

	dm.KichHoat = body.KichHoat
	config.DB.Save(&dm)

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật trạng thái thành công",
		"data":    dm,
	})
}

// ✅ Xem chi tiết danh mục
func GetDanhMucByID(c *gin.Context) {
	role, _ := c.Get("vai_tro")

	id := c.Param("id")
	var danhMuc models.DanhMuc
	if err := config.DB.First(&danhMuc, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy danh mục"})
		return
	}

	// Người dùng thường chỉ xem được danh mục đã kích hoạt
	if role != "admin" && !danhMuc.KichHoat {
		c.JSON(http.StatusForbidden, gin.H{"error": "Danh mục này chưa được kích hoạt"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": danhMuc})
}
