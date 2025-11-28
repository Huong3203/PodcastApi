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
	SessionID string `json:"session_id" binding:"required"`
}

func LoginWithClerk(c *gin.Context) {

	fmt.Println("🔵 [LoginWithClerk] Bắt đầu đăng nhập Clerk")

	var input ClerkLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id bắt buộc"})
		return
	}

	// 1. Get session từ Clerk
	sess, err := middleware.ClerkClient.Sessions().Read(input.SessionID)
	if err != nil {
		fmt.Println("❌ Clerk session error:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session không hợp lệ"})
		return
	}

	// 2. Lấy user từ Clerk
	clerkUser, err := middleware.ClerkClient.Users().Read(sess.UserID)
	if err != nil {
		fmt.Println("❌ Clerk user error:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không lấy được user từ Clerk"})
		return
	}

	// 3. Lấy email
	email := ""
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	// 4. Lấy tên + avatar
	hoTen := ""
	if clerkUser.FirstName != nil {
		hoTen += *clerkUser.FirstName
	}
	if clerkUser.LastName != nil {
		hoTen += " " + *clerkUser.LastName
	}

	avatar := clerkUser.ProfileImageURL

	// 5. Kiểm tra / tạo user trong DB
	var user models.NguoiDung
	result := config.DB.Where("id = ?", clerkUser.ID).First(&user)

	if result.Error != nil {
		fmt.Println("ℹ User chưa tồn tại → tạo mới")
		user = models.NguoiDung{
			ID:       clerkUser.ID,
			Email:    email,
			HoTen:    hoTen,
			Avatar:   avatar,
			VaiTro:   "user",
			KichHoat: true,
			Provider: "clerk",
		}
		if err := config.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo user mới"})
			return
		}
	}

	// 6. Tạo JWT (đã sửa lỗi tham số)
	token, err := utils.GenerateToken(user.ID, user.VaiTro)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo token"})
		return
	}

	// 7. Trả kết quả
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":      user.ID,
			"email":   user.Email,
			"ho_ten":  user.HoTen,
			"avatar":  user.Avatar,
			"vai_tro": user.VaiTro,
		},
		"token": token,
	})
}
