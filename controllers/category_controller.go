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

// đây là controller để trả về danh sách danh mục: phân trang, tìm kiếm, lọc trạng thái,
func GetDanhMucs(c *gin.Context) {
	var danhMucs []models.DanhMuc
	var total int64

	// Phân trang
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Tìm kiếm & lọc
	search := c.Query("search")
	status := c.Query("status") // "true"/"false"

	query := config.DB.Model(&models.DanhMuc{})

	// Lấy role từ context (giao sử đã có middleware đã set)
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		query = query.Where("kich_hoat = ?", true) // chỉ lấy danh mục đã kích hoạt
	}

	if search != "" {
		query = query.Where("LOWER(ten_danh_muc) LIKE ?", "%"+strings.ToLower(search)+"%")
	}
	if status != "" && role == "admin" {
		switch status {
		case "true":
			query = query.Where("kich_hoat = ?", true)
		case "false":
			query = query.Where("kich_hoat = ?", false)
		}
	}

	// Đếm tổng bản ghi
	query.Count(&total)

	// Lấy dữ liệu
	query.Offset(offset).Limit(limit).Find(&danhMucs)

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

func ToggleDanhMucStatus(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền thực hiện hành động này"})
		return
	}
	id := c.Param("id")
	var body struct {
		KichHoat bool `json:"kich_hoat"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Body không hợp lệ"})
		return
	}

	var dm models.DanhMuc
	if err := config.DB.First(&dm, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy danh mục"})
		return
	}

	dm.KichHoat = body.KichHoat
	config.DB.Save(&dm)

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật trạng thái thành công", "data": dm})
}

func CreateDanhMuc(c *gin.Context) {
	if role, _ := c.Get("vai_tro"); role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền thực hiện hành động này"})
		return
	}
	var input struct {
		TenDanhMuc string `json:"ten_danh_muc" binding:"required"`
		MoTa       string `json:"mo_ta"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Body không hợp lệ"})
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

	c.JSON(http.StatusCreated, gin.H{"message": "Tạo danh mục thành công", "data": danhMuc})
}

func UpdateDanhMuc(c *gin.Context) {
	if role, _ := c.Get("vai_tro"); role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền thực hiện hành động này"})
		return
	}
	var input struct {
		TenDanhMuc string `json:"ten_danh_muc" binding:"required"`
		MoTa       string `json:"mo_ta"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Body không hợp lệ"})
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

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật danh mục thành công", "data": danhMuc})
}

func GetDanhMucByID(c *gin.Context) {
	if role, _ := c.Get("vai_tro"); role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền thực hiện hành động này"})
		return
	}
	id := c.Param("id")
	var danhMuc models.DanhMuc

	if err := config.DB.First(&danhMuc, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy danh mục"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": danhMuc})
}
