package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// GET /api/users/profile
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

// PUT /api/users/profile
type UpdateProfileInput struct {
	HoTen string `json:"ho_ten" binding:"required"`
	Email string `json:"email" binding:"required,email"`
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

	// Tiến hành cập nhật và kiểm tra có user nào bị ảnh hưởng không
	tx := config.DB.Model(&models.NguoiDung{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"ho_ten": input.HoTen,
			"email":  input.Email,
		})

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

// POST /api/users/change-password
type ChangePasswordInput struct {
	MatKhauCu  string `json:"mat_khau_cu" binding:"required"`
	MatKhauMoi string `json:"mat_khau_moi" binding:"required,min=6"`
}

func ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")

	var input ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.NguoiDung
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}

	// Kiểm tra mật khẩu cũ
	if err := bcrypt.CompareHashAndPassword([]byte(user.MatKhau), []byte(input.MatKhauCu)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Mật khẩu cũ không đúng"})
		return
	}

	// Hash mật khẩu mới
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.MatKhauMoi), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mã hoá mật khẩu"})
		return
	}

	// Cập nhật mật khẩu
	if err := config.DB.Model(&user).Update("mat_khau", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đổi mật khẩu thất bại"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đổi mật khẩu thành công"})
}
