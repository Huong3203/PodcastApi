package controllers

import (
	"fmt"
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/middleware"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
)

type ClerkLoginInput struct {
	SessionToken string `json:"session_token" binding:"required"`
}

func LoginWithClerk(c *gin.Context) {

	var input ClerkLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_token bắt buộc"})
		return
	}

	fmt.Println("📥 session_token:", input.SessionToken)

	// 1. Verify session token → get session
	sess, err := middleware.ClerkClient.Sessions().VerifyToken(input.SessionToken)
	if err != nil {
		fmt.Println("❌ Clerk verify lỗi:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
		return
	}

	fmt.Println("✅ Verify token OK — Session:", sess.ID, "User:", sess.UserID)

	// 2. Lấy user
	clerkUser, err := middleware.ClerkClient.Users().Read(sess.UserID)
	if err != nil {
		fmt.Println("❌ Lỗi lấy user:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không lấy được user"})
		return
	}

	email := ""
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	fullName := ""
	if clerkUser.FirstName != nil && clerkUser.LastName != nil {
		fullName = *clerkUser.FirstName + " " + *clerkUser.LastName
	}

	avatar := clerkUser.ProfileImageURL

	// 3. Tạo user nếu chưa có
	var user models.NguoiDung
	result := config.DB.Where("id = ?", clerkUser.ID).First(&user)

	if result.Error != nil {
		user = models.NguoiDung{
			ID:       clerkUser.ID,
			Email:    email,
			HoTen:    fullName,
			Avatar:   avatar,
			VaiTro:   "user",
			KichHoat: true,
			Provider: "clerk",
		}
		config.DB.Create(&user)
	}

	// 4. Tạo JWT
	token, err := utils.GenerateToken(user.ID, user.VaiTro, "clerk")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tạo được token"})
		return
	}

	// 5. Trả về
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
