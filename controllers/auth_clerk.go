package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
)

// 🔹 Struct nhận Session Token từ Clerk
type ClerkLoginInput struct {
	SessionToken string `json:"session_token" binding:"required"`
}

// 🔹 Clerk User Response Structure
type ClerkUserResponse struct {
	ID             string `json:"id"`
	EmailAddresses []struct {
		EmailAddress string `json:"email_address"`
	} `json:"email_addresses"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	ImageURL  string  `json:"image_url"`
}

// 🔹 Handler: Login với Clerk Session Token
func ClerkLogin(c *gin.Context) {
	var input ClerkLoginInput

	// ✅ Parse JSON từ frontend
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu session_token"})
		return
	}

	// ✅ Verify session token và lấy user info từ Clerk
	clerkUser, err := verifyClerkToken(input.SessionToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session token không hợp lệ"})
		return
	}

	// ✅ Lấy thông tin cơ bản
	clerkUserID := clerkUser.ID
	email := ""
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	fullName := ""
	if clerkUser.FirstName != nil && clerkUser.LastName != nil {
		fullName = *clerkUser.FirstName + " " + *clerkUser.LastName
	} else if clerkUser.FirstName != nil {
		fullName = *clerkUser.FirstName
	}

	avatar := clerkUser.ImageURL

	// ✅ Validate dữ liệu cơ bản
	if email == "" || clerkUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu thông tin từ Clerk"})
		return
	}

	// ✅ Fallback tên nếu rỗng
	if fullName == "" {
		fullName = "User"
	}

	// ✅ Tìm user trong DB theo ID hoặc email
	var user models.NguoiDung
	err = config.DB.Where("id = ? OR email = ?", clerkUserID, email).First(&user).Error

	if err != nil {
		// ✅ User chưa tồn tại → Tạo mới
		user = models.NguoiDung{
			ID:       clerkUserID,
			Email:    email,
			HoTen:    fullName,
			Avatar:   avatar,
			VaiTro:   "user",
			Provider: "clerk",
			KichHoat: true,
		}
		if err := config.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể tạo tài khoản",
			})
			return
		}
	} else {
		// ✅ User đã tồn tại → Cập nhật thông tin
		user.HoTen = fullName
		user.Avatar = avatar
		user.Provider = "clerk"

		if err := config.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể cập nhật thông tin",
			})
			return
		}
	}

	// ✅ Kiểm tra tài khoản có bị khóa không
	if !user.KichHoat {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Tài khoản của bạn đã bị tạm khóa",
		})
		return
	}

	// ✅ Tạo JWT token local
	localToken, err := utils.GenerateToken(user.ID, user.VaiTro)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể tạo token",
		})
		return
	}

	// ✅ Trả về token + user info
	c.JSON(http.StatusOK, gin.H{
		"token": localToken,
		"user": gin.H{
			"id":      user.ID,
			"email":   user.Email,
			"ho_ten":  user.HoTen,
			"vai_tro": user.VaiTro,
			"avatar":  user.Avatar,
			"vip":     user.VIP,
		},
	})
}

// ✅ Helper function: Verify Clerk token và lấy user info
func verifyClerkToken(sessionToken string) (*ClerkUserResponse, error) {
	clerkSecretKey := os.Getenv("CLERK_SECRET_KEY")
	if clerkSecretKey == "" {
		return nil, fmt.Errorf("CLERK_SECRET_KEY không được thiết lập")
	}

	// ✅ Gọi Clerk API để verify token và lấy user info
	// Clerk sử dụng session token trong header để authenticate
	req, err := http.NewRequest("GET", "https://api.clerk.com/v1/me", nil)
	if err != nil {
		return nil, err
	}

	// ✅ Set authorization header với session token
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid session token: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var clerkUser ClerkUserResponse
	if err := json.Unmarshal(body, &clerkUser); err != nil {
		return nil, err
	}

	return &clerkUser, nil
}
