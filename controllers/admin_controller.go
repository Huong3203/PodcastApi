package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
)

// ==========================
// Thống kê admin
// ==========================
func GetAdminStats(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền truy cập"})
		return
	}

	db := config.DB
	var userCount int64
	db.Model(&models.NguoiDung{}).Count(&userCount)

	c.JSON(http.StatusOK, gin.H{"total_users": userCount})
}

// ==========================
// Quản lý user: danh sách, đổi role, khóa/kích hoạt
// ==========================
func GetAllUsers(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền truy cập"})
		return
	}

	var users []models.NguoiDung
	if err := config.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách người dùng"})
		return
	}

	for i := range users {
		users[i].MatKhau = ""
	}

	c.JSON(http.StatusOK, gin.H{"total": len(users), "users": users})
}

func UpdateUserRole(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền đổi vai trò"})
		return
	}

	id := c.Param("id")
	var input struct {
		VaiTro string `json:"vai_tro"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	if input.VaiTro != "admin" && input.VaiTro != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vai trò không hợp lệ"})
		return
	}

	if err := config.DB.Model(&models.NguoiDung{}).Where("id = ?", id).Update("vai_tro", input.VaiTro).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật vai trò"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật vai trò thành công"})
}

func ToggleUserActivation(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền cập nhật trạng thái"})
		return
	}

	id := c.Param("id")
	var user models.NguoiDung
	if err := config.DB.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}

	newStatus := !user.KichHoat
	if err := config.DB.Model(&user).Update("kich_hoat", newStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật trạng thái thành công", "kich_hoat": newStatus})
}
