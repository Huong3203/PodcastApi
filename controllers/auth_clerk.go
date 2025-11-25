package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/middleware"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"

	"github.com/gin-gonic/gin"
)

// Input khi client gửi session_id từ Clerk/Google
type ClerkLoginInput struct {
	SessionID string `json:"session_id" binding:"required"`
}

// API đăng nhập với Clerk
func LoginWithClerk(c *gin.Context) {
	var input ClerkLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id bắt buộc"})
		return
	}

	// 1. Lấy session từ Clerk
	sess, err := middleware.ClerkClient.Sessions().Read(input.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session không hợp lệ"})
		return
	}

	// 2. Lấy user từ session
	clerkUser, err := middleware.ClerkClient.Users().Read(sess.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không lấy được user từ Clerk"})
		return
	}

	// 3. Lấy email
	email := ""
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	// 4. Lấy tên đầy đủ và avatar
	hoTen := ""
	if clerkUser.FirstName != nil && clerkUser.LastName != nil {
		hoTen = *clerkUser.FirstName + " " + *clerkUser.LastName
	}

	avatar := ""
	if clerkUser.ProfileImageURL != "" {
		avatar = clerkUser.ProfileImageURL
	}

	// 5. Lưu user vào DB nếu chưa có
	var user models.NguoiDung
	result := config.DB.Where("id = ?", clerkUser.ID).First(&user)
	if result.Error != nil {
		user = models.NguoiDung{
			ID:       clerkUser.ID,
			Email:    email,
			VaiTro:   "user",
			KichHoat: true,
			Provider: "clerk",
			HoTen:    hoTen,
			Avatar:   avatar,
		}
		config.DB.Create(&user)
	}

	// 6. Tạo token JWT
	token, err := utils.GenerateToken(user.ID, user.VaiTro, "clerk")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tạo được token"})
		return
	}

	// 7. Trả về client
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":      user.ID,
			"email":   user.Email,
			"ho_ten":  user.HoTen,
			"vai_tro": user.VaiTro,
			"avatar":  user.Avatar,
		},
		"token": token,
	})
}
