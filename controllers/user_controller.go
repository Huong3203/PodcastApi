package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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

// ==========================
// 🔹 POST /api/users/change-password
// ==========================
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

// ==========================
// 🔹 GET /api/admin/users
// ==========================
// Lấy tất cả user (chỉ admin)
func GetAllUsers(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền truy cập"})
		return
	}

	db := config.DB
	var users []models.NguoiDung

	if err := db.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách người dùng"})
		return
	}

	// Ẩn mật khẩu trước khi trả
	for i := range users {
		users[i].MatKhau = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(users),
		"users": users,
	})
}
