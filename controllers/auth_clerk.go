package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Input khi client gửi idToken từ Clerk/Google
type ClerkLoginInput struct {
	IDToken string `json:"id_token" binding:"required"`
	Email   string `json:"email" binding:"required,email"`
	HoTen   string `json:"ho_ten"`
	Avatar  string `json:"avatar"`
}

// API đăng nhập với Google/Clerk
func LoginWithClerk(c *gin.Context) {
	var input ClerkLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Validate idToken với Clerk API nếu cần (verify token)
	// Ví dụ: utils.VerifyClerkToken(input.IDToken)

	var user models.NguoiDung
	err := config.DB.Where("email = ? AND provider = ?", input.Email, "clerk").First(&user).Error
	if err != nil {
		// Người dùng chưa tồn tại, tạo mới
		newUser := models.NguoiDung{
			ID:       uuid.New().String(),
			Email:    input.Email,
			HoTen:    input.HoTen,
			Avatar:   input.Avatar,
			VaiTro:   "user",
			KichHoat: true,
			Provider: "clerk",
		}

		if err := config.DB.Create(&newUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tạo người dùng"})
			return
		}

		user = newUser
	}

	// Tạo token JWT
	token, _ := utils.GenerateToken(user.ID, user.VaiTro, "clerk")
	user.MatKhau = "" // ẩn mật khẩu

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"token":    token,
		"provider": "clerk",
	})
}
