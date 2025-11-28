package controllers

import (
	"fmt"
	"net/http"
	"time"

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

	// 1. Lấy session từ Clerk
	sess, err := middleware.ClerkClient.Sessions().Read(input.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session không hợp lệ"})
		return
	}

	// 2. Lấy user từ Clerk
	clerkUser, err := middleware.ClerkClient.Users().Read(sess.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không lấy được user từ Clerk"})
		return
	}

	// 3. Lấy email, tên, avatar
	email := ""
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	hoTen := ""
	if clerkUser.FirstName != nil {
		hoTen += *clerkUser.FirstName
	}
	if clerkUser.LastName != nil {
		if hoTen != "" {
			hoTen += " "
		}
		hoTen += *clerkUser.LastName
	}

	avatar := clerkUser.ProfileImageURL

	// 4. Kiểm tra user theo email
	var user models.NguoiDung
	err = config.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		if err != nil && !IsRecordNotFound(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi DB"})
			return
		}

		// User chưa tồn tại → tạo mới
		fmt.Println("ℹ User chưa tồn tại → tạo mới")
		user = models.NguoiDung{
			ID:       clerkUser.ID, // dùng ID Clerk
			Email:    email,
			HoTen:    hoTen,
			Avatar:   avatar,
			VaiTro:   "user",
			KichHoat: true,
			Provider: "clerk",
			NgayTao:  time.Now(),
		}

		if err := config.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  "Không thể tạo user mới",
				"detail": err.Error(),
			})
			return
		}
	} else {
		// User đã tồn tại → update tên + avatar
		user.HoTen = hoTen
		user.Avatar = avatar
		config.DB.Save(&user)
	}

	// 5. Tạo JWT
	token, err := utils.GenerateToken(user.ID, user.VaiTro)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo token"})
		return
	}

	// 6. Trả kết quả
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

// helper kiểm tra RecordNotFound
func IsRecordNotFound(err error) bool {
	// Nếu dùng GORM v2
	return err != nil && err.Error() == "record not found"
}
