package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
)

// ==========================
// 🔹 GET /api/users/profile
// ==========================
func GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var user models.NguoiDung
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}

	user.MatKhau = ""
	c.JSON(http.StatusOK, user)
}

// ==========================
// 🔹 PUT /api/users/profile
// ==========================
type UpdateProfileInput struct {
	HoTen  string `json:"ho_ten" binding:"required"`
	Email  string `json:"email" binding:"required,email"`
	Avatar string `json:"avatar"` // ✅ avatar, không bắt buộc
}

func UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var input UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Kiểm tra email đã tồn tại (trừ user hiện tại)
	var existingUser models.NguoiDung
	if err := config.DB.
		Where("email = ? AND id != ?", input.Email, userID).
		First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email đã được sử dụng"})
		return
	}

	// Tạo map cập nhật
	updateData := map[string]interface{}{
		"ho_ten": input.HoTen,
		"email":  input.Email,
	}
	if input.Avatar != "" {
		updateData["avatar"] = input.Avatar
	}

	tx := config.DB.Model(&models.NguoiDung{}).
		Where("id = ?", userID).
		Updates(updateData)

	if tx.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}

	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cập nhật thất bại"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật thành công"})
}
