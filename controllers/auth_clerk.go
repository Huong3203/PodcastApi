package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/middleware"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
)

type ClerkLoginInput struct {
	SessionID string `json:"session_id" binding:"required"`
	Email     string `json:"email" binding:"required"`
	HoTen     string `json:"ho_ten"`
	Avatar    string `json:"avatar"`
}

func LoginWithClerk(c *gin.Context) {
	var input ClerkLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu dữ liệu"})
		return
	}

	sess, err := middleware.ClerkClient.Sessions().Read(input.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session không hợp lệ"})
		return
	}

	clerkUser, err := middleware.ClerkClient.Users().Read(sess.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không lấy được thông tin user"})
		return
	}

	email := input.Email
	if email == "" && len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	hoTen := input.HoTen
	if hoTen == "" {
		if clerkUser.FirstName != nil {
			hoTen = *clerkUser.FirstName
		}
		if clerkUser.LastName != nil {
			hoTen += " " + *clerkUser.LastName
		}
	}

	avatar := input.Avatar
	if avatar == "" {
		avatar = clerkUser.ProfileImageURL
	}

	var user models.NguoiDung
	err = config.DB.First(&user, "email = ?", email).Error
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
