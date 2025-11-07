package controllers

import (
	"net/http"
	"os"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"

	"github.com/clerkinc/clerk-sdk-go/clerk"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var clerkClient clerk.Client

// ✅ Khởi tạo Clerk
func InitClerk() {
	secretKey := os.Getenv("CLERK_SECRET_KEY")
	if secretKey == "" {
		panic("CLERK_SECRET_KEY không tồn tại trong ENV")
	}

	var err error
	clerkClient, err = clerk.NewClient(secretKey)
	if err != nil {
		panic("Không thể tạo Clerk client: " + err.Error())
	}
}

type ClerkLoginInput struct {
	ClerkToken string `json:"clerk_token" binding:"required"`
}

// API đăng nhập bằng Clerk
func ClerkLogin(c *gin.Context) {
	var input ClerkLoginInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu clerk_token"})
		return
	}

	// Verify session token từ Clerk
	session, err := clerkClient.Sessions().Verify(input.ClerkToken)
	if err != nil || session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
		return
	}

	userID := session.UserID

	// Lấy dữ liệu user từ Clerk
	clerkUser, err := clerkClient.Users().Read(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc thông tin user từ Clerk"})
		return
	}

	email := clerkUser.EmailAddresses[0].EmailAddress
	fullName := clerkUser.FirstName + " " + clerkUser.LastName
	avatar := clerkUser.ProfileImageURL

	// Sync xuống MySQL
	var user models.NguoiDung
	if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
		user = models.NguoiDung{
			ID:       uuid.New().String(),
			Email:    email,
			HoTen:    fullName,
			VaiTro:   "user",
			Avatar:   avatar,
			KichHoat: true,
		}

		if err := config.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo user mới"})
			return
		}
	}

	// ✅ Tạo JWT riêng
	token, err := utils.GenerateToken(user.ID, user.VaiTro)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo JWT"})
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
