package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/middleware"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
)

// 🔹 Struct nhận dữ liệu từ frontend khi login Google qua Clerk
type ClerkGoogleLoginInput struct {
	IDToken string `json:"id_token" binding:"required"` // ✅ Nhận token từ Clerk (frontend gửi)
	Email   string `json:"email" binding:"required"`
	HoTen   string `json:"ho_ten"` // Tên người dùng (optional)
	Avatar  string `json:"avatar"` // Avatar URL (optional)
}

// 🔹 Handler: Login Google thông qua Clerk
func LoginWithClerkGoogle(c *gin.Context) {
	var input ClerkGoogleLoginInput

	// ✅ Parse JSON từ frontend
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu dữ liệu bắt buộc (id_token hoặc email)"})
		return
	}

	// ✅ Verify token từ Clerk bằng Sessions().Read
	// input.IDToken chính là sessionToken từ Clerk
	session, err := middleware.ClerkClient.Sessions().Read(input.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token Clerk không hợp lệ hoặc đã hết hạn"})
		return
	}

	// ✅ Lấy thông tin user từ Clerk
	clerkUserID := session.UserID
	clerkUser, err := middleware.ClerkClient.Users().Read(clerkUserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không lấy được thông tin user từ Clerk"})
		return
	}

	// ✅ Xử lý email (ưu tiên input, fallback Clerk)
	email := input.Email
	if email == "" && len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy email"})
		return
	}

	// ✅ Xử lý tên (ưu tiên input, fallback Clerk)
	hoTen := input.HoTen
	if hoTen == "" {
		if clerkUser.FirstName != nil {
			hoTen = *clerkUser.FirstName
		}
		if clerkUser.LastName != nil {
			if hoTen != "" {
				hoTen += " "
			}
			hoTen += *clerkUser.LastName
		}
	}

	// ✅ Xử lý avatar (ưu tiên input, fallback Clerk)
	avatar := input.Avatar
	if avatar == "" {
		avatar = clerkUser.ProfileImageURL
	}

	// ✅ Tìm hoặc tạo user trong DB
	var user models.NguoiDung
	err = config.DB.First(&user, "email = ?", email).Error
	if err != nil {
		// ✅ User chưa tồn tại → Tạo mới
		user = models.NguoiDung{
			ID:       clerkUser.ID, // Dùng Clerk ID làm primary key
			Email:    email,
			HoTen:    hoTen,
			Avatar:   avatar,
			VaiTro:   "user",  // Role mặc định
			Provider: "clerk", // Đánh dấu đăng nhập qua Clerk
			KichHoat: true,    // Tài khoản active
		}
		if err := config.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo tài khoản"})
			return
		}
	} else {
		// ✅ User đã tồn tại → Cập nhật thông tin
		user.HoTen = hoTen
		user.Avatar = avatar
		if err := config.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật thông tin"})
			return
		}
	}

	// ✅ Tạo JWT token local để frontend sử dụng
	token, err := utils.GenerateToken(user.ID, user.VaiTro)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo token"})
		return
	}

	// ✅ Trả về token + user info
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":      user.ID,
			"email":   user.Email,
			"ho_ten":  user.HoTen,
			"vai_tro": user.VaiTro,
			"avatar":  user.Avatar,
		},
	})
}
