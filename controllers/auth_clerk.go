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

// ==========================
// Clerk Login
// ==========================
type ClerkLoginInput struct {
	ClerkToken string `json:"clerk_token" binding:"required"`
}

func ClerkLogin(c *gin.Context) {
	var input ClerkLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu clerk_token"})
		return
	}

	session, err := clerkClient.Sessions().Verify(input.ClerkToken, "")
	if err != nil || session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
		return
	}

	userID := session.UserID
	clerkUser, _ := clerkClient.Users().Read(userID)

	email := clerkUser.EmailAddresses[0].EmailAddress
	fullName := ""
	if clerkUser.FirstName != nil {
		fullName += *clerkUser.FirstName
	}
	if clerkUser.LastName != nil {
		fullName += " " + *clerkUser.LastName
	}
	if fullName == " " {
		fullName = "Người dùng"
	}
	avatar := clerkUser.ProfileImageURL

	var user models.NguoiDung
	if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
		user = models.NguoiDung{
			ID:       uuid.New().String(),
			Email:    email,
			MatKhau:  "clerk",
			HoTen:    fullName,
			VaiTro:   "user",
			KichHoat: true,
			Provider: "clerk",
			Avatar:   avatar,
		}
		config.DB.Create(&user)
	}

	token, _ := utils.GenerateToken(user.ID, user.VaiTro, user.Provider)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}
