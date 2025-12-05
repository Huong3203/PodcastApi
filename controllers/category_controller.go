package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
)

//
// PUBLIC
//

// Lấy danh sách danh mục (active + phân trang + search)
func GetDanhMucs(c *gin.Context) {
	var danhMucs []models.DanhMuc
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	search := c.Query("search")

	query := config.DB.Model(&models.DanhMuc{}).Where("kich_hoat = ?", true)

	if search != "" {
		query = query.Where("LOWER(ten_danh_muc) LIKE ?", "%"+strings.ToLower(search)+"%")
	}

	query.Count(&total)
	query.Offset(offset).Limit(limit).Order("ngay_tao DESC").Find(&danhMucs)

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

// Xem chi tiết
func GetDanhMucByID(c *gin.Context) {
	role, _ := c.Get("vai_tro")

	id := c.Param("id")
	var danhMuc models.DanhMuc

	if err := config.DB.First(&danhMuc, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy danh mục"})
		return
	}

	if role != "admin" && !danhMuc.KichHoat {
		c.JSON(http.StatusForbidden, gin.H{"error": "Danh mục chưa được kích hoạt"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": danhMuc})
}

//
// ADMIN
//

// Tạo danh mục
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

	danhMuc := models.DanhMuc{
		ID:         uuid.New().String(),
		TenDanhMuc: input.TenDanhMuc,
		Slug:       slug.Make(input.TenDanhMuc),
		MoTa:       input.MoTa,
		KichHoat:   true,
	}

	if err := config.DB.Create(&danhMuc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo danh mục"})
		return
	}

	// 🔔 Tạo thông báo
	_ = services.CreateNotification(
		c.GetString("user_id"),
		danhMuc.ID,
		"create_category",
		fmt.Sprintf("Danh mục '%s' vừa được tạo", danhMuc.TenDanhMuc),
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tạo danh mục thành công",
		"data":    danhMuc,
	})
}

// Cập nhật danh mục
func UpdateDanhMuc(c *gin.Context) {
	if role, _ := c.Get("vai_tro"); role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền cập nhật"})
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

	var dm models.DanhMuc
	if err := config.DB.First(&dm, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy"})
		return
	}

	dm.TenDanhMuc = input.TenDanhMuc
	dm.MoTa = input.MoTa
	dm.Slug = slug.Make(input.TenDanhMuc)

	config.DB.Save(&dm)

	// 🔔 Notification
	_ = services.CreateNotification(
		c.GetString("user_id"),
		dm.ID,
		"update_category",
		fmt.Sprintf("Danh mục '%s' vừa được cập nhật", dm.TenDanhMuc),
	)

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật thành công", "data": dm})
}

// Bật/Tắt danh mục
func ToggleDanhMucStatus(c *gin.Context) {
	if role, _ := c.Get("vai_tro"); role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không có quyền"})
		return
	}

	var body struct {
		KichHoat bool `json:"kich_hoat"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	id := c.Param("id")

	var dm models.DanhMuc
	if err := config.DB.First(&dm, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy"})
		return
	}

	dm.KichHoat = body.KichHoat
	config.DB.Save(&dm)

	status := "tắt"
	if dm.KichHoat {
		status = "bật"
	}

	_ = services.CreateNotification(
		c.GetString("user_id"),
		dm.ID,
		"toggle_category",
		fmt.Sprintf("Danh mục '%s' vừa được %s", dm.TenDanhMuc, status),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật trạng thái thành công",
		"data":    dm,
	})
}
