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

func InitClerk() {
	secretKey := os.Getenv("CLERK_SECRET_KEY")
	if secretKey == "" {
		panic("CLERK_SECRET_KEY không tồn tại trong ENV")
	}

	client, err := clerk.NewClient(secretKey)
	if err != nil {
		panic("Clerk init failed: " + err.Error())
	}

	clerkClient = client
}

type ClerkLoginInput struct {
	ClerkToken string `json:"clerk_token" binding:"required"`
}

func ClerkLogin(c *gin.Context) {
	var input ClerkLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu clerk_token"})
		return
	}

	// Verify token Clerk
	session, err := clerkClient.Sessions().Verify(input.ClerkToken, "")
	if err != nil || session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
		return
	}

	// Lấy user info từ Clerk
	clerkUser, err := clerkClient.Users().Read(session.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc thông tin user từ Clerk"})
		return
	}

	email := ""
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	fn, ln := "", ""
	if clerkUser.FirstName != nil {
		fn = *clerkUser.FirstName
	}
	if clerkUser.LastName != nil {
		ln = *clerkUser.LastName
	}

	fullName := fn + " " + ln
	if fullName == " " {
		fullName = "Người dùng"
	}

	avatar := clerkUser.ProfileImageURL

	// Kiểm tra user đã có trong DB chưa
	var user models.NguoiDung
	result := config.DB.Where("email = ? AND provider = ?", email, "clerk").First(&user)

	if result.Error != nil || result.RowsAffected == 0 {
		// Tạo mới nếu chưa có
		user = models.NguoiDung{
			ID:       uuid.New().String(),
			Email:    email,
			MatKhau:  "clerk", // placeholder, không dùng
			HoTen:    fullName,
			VaiTro:   "user",
			Avatar:   avatar,
			KichHoat: true,
			Provider: "clerk",
		}
		if err := config.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo user mới"})
			return
		}
	}

	// Tạo token backend
	token, err := utils.GenerateToken(user.ID, user.VaiTro, "clerk")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"token":    token,
		"provider": "clerk",
	})
}
