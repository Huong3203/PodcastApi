package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/middleware"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
)

// ✅ Input nhận từ frontend
type ClerkLoginInput struct {
	SessionID string `json:"session_id" binding:"required"` // Clerk session ID
	Email     string `json:"email" binding:"required"`      // Email từ Clerk
	HoTen     string `json:"ho_ten"`                        // Optional
	Avatar    string `json:"avatar"`                        // Optional
}

func LoginWithClerk(c *gin.Context) {
	var input ClerkLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu dữ liệu"})
		return
	}

	// Verify session với Clerk
	sess, err := middleware.ClerkClient.Sessions().Read(input.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session không hợp lệ"})
		return
	}

	// Lấy user từ Clerk
	clerkUser, err := middleware.ClerkClient.Users().Read(sess.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không lấy được thông tin user"})
		return
	}

	email := ""
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	// Tên
	hoTen := input.HoTen
	if hoTen == "" {
		if clerkUser.FirstName != nil {
			hoTen = *clerkUser.FirstName
		}
		if clerkUser.LastName != nil {
			hoTen += " " + *clerkUser.LastName
		}
	}

	// Avatar
	avatar := input.Avatar
	if avatar == "" {
		avatar = clerkUser.ProfileImageURL
	}

	// Lưu hoặc lấy user trong DB
	var user models.NguoiDung
	err = config.DB.Where("email = ?", email).First(&user).Error

	if err != nil {
		user = models.NguoiDung{
			ID:       clerkUser.ID,
			Email:    email,
			HoTen:    hoTen,
			Avatar:   avatar,
			VaiTro:   "user",
			Provider: "clerk",
			KichHoat: true,
		}
		config.DB.Create(&user)
	} else {
		user.HoTen = hoTen
		user.Avatar = avatar
		config.DB.Save(&user)
	}

	// Tạo JWT local
	token, err := utils.GenerateToken(user.ID, user.VaiTro)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  user,
		"token": token,
	})
}

// Helper kiểm tra RecordNotFound
func IsRecordNotFound(err error) bool {
	// GORM v2
	return err != nil && err.Error() == "record not found"
}
