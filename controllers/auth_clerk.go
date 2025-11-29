package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
)

// 🔹 Struct nhận dữ liệu từ frontend sau khi Clerk đã verify
type ClerkGoogleLoginInput struct {
	ClerkUserID string `json:"clerk_user_id" binding:"required"` // User ID từ Clerk
	Email       string `json:"email" binding:"required"`
	HoTen       string `json:"ho_ten"` // Tên người dùng
	Avatar      string `json:"avatar"` // Avatar URL
}

// 🔹 Handler: Login Google thông qua Clerk (Simplified)
func LoginWithClerkGoogle(c *gin.Context) {
	var input ClerkGoogleLoginInput

	// ✅ Parse JSON từ frontend
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu dữ liệu bắt buộc"})
		return
	}

	// ✅ Validate dữ liệu cơ bản
	if input.ClerkUserID == "" || input.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu clerk_user_id hoặc email"})
		return
	}

	// ✅ Fallback tên nếu rỗng
	if input.HoTen == "" {
		input.HoTen = "User"
	}

	// ✅ Tìm hoặc tạo user trong DB
	var user models.NguoiDung
	err := config.DB.First(&user, "id = ?", input.ClerkUserID).Error
	if err != nil {
		// ✅ User chưa tồn tại → Tạo mới
		user = models.NguoiDung{
			ID:       input.ClerkUserID, // Dùng Clerk ID làm primary key
			Email:    input.Email,
			HoTen:    input.HoTen,
			Avatar:   input.Avatar,
			VaiTro:   "user",  // Role mặc định
			Provider: "clerk", // Đánh dấu đăng nhập qua Clerk
			KichHoat: true,    // Tài khoản active
		}
		if err := config.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể tạo tài khoản",
			})
			return
		}
	} else {
		// ✅ User đã tồn tại → Cập nhật thông tin
		user.HoTen = input.HoTen
		user.Avatar = input.Avatar
		user.Email = input.Email
		if err := config.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể cập nhật thông tin",
			})
			return
		}
	}

	// ✅ Tạo JWT token local để frontend sử dụng
	localToken, err := utils.GenerateToken(user.ID, user.VaiTro)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể tạo token",
		})
		return
	}

	// ✅ Trả về token + user info
	c.JSON(http.StatusOK, gin.H{
		"token": localToken,
		"user": gin.H{
			"id":      user.ID,
			"email":   user.Email,
			"ho_ten":  user.HoTen,
			"vai_tro": user.VaiTro,
			"avatar":  user.Avatar,
		},
	})
}
