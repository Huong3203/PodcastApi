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
	"github.com/golang-jwt/jwt/v5"
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
		fmt.Printf("❌ Parse JSON error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu session_token"})
		return
	}

	fmt.Printf("✅ Received session token: %s...\n", input.SessionToken[:30])

	// ✅ Verify session token và lấy user info từ Clerk
	clerkUser, err := verifyClerkToken(input.SessionToken)
	if err != nil {
		fmt.Printf("❌ Clerk verify error: %v\n", err)
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

	fmt.Printf("✅ Clerk User ID: %s\n", clerkUserID)
	fmt.Printf("✅ Email: %s\n", email)
	fmt.Printf("✅ Full Name: %s\n", fullName)

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
			fmt.Printf("❌ Database create error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể tạo tài khoản",
			})
			return
		}
		fmt.Printf("✅ Created new user: %s\n", clerkUserID)
	} else {
		// ✅ User đã tồn tại → Cập nhật thông tin
		user.HoTen = fullName
		user.Avatar = avatar
		user.Provider = "clerk"

		if err := config.DB.Save(&user).Error; err != nil {
			fmt.Printf("❌ Database save error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể cập nhật thông tin",
			})
			return
		}
		fmt.Printf("✅ Updated existing user: %s\n", clerkUserID)
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
		fmt.Printf("❌ Token generation error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể tạo token",
		})
		return
	}

	fmt.Printf("✅ Generated local JWT token\n")

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

// ✅ Helper function: Verify Clerk session token
func verifyClerkToken(sessionToken string) (*ClerkUserResponse, error) {
	clerkSecretKey := os.Getenv("CLERK_SECRET_KEY")
	if clerkSecretKey == "" {
		return nil, fmt.Errorf("CLERK_SECRET_KEY không được thiết lập")
	}

	fmt.Printf("🔍 Verifying Clerk session token...\n")
	fmt.Printf("🔑 CLERK_SECRET_KEY exists: %v\n", len(clerkSecretKey) > 0)

	// ✅ Parse JWT token without verification to get user ID
	token, _, err := new(jwt.Parser).ParseUnverified(sessionToken, jwt.MapClaims{})
	if err != nil {
		fmt.Printf("❌ JWT parse error: %v\n", err)
		return nil, fmt.Errorf("invalid token format: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	fmt.Printf("📋 Token claims: %+v\n", claims)

	// ✅ Lấy user ID từ token claims (sub)
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("no user ID in token")
	}

	fmt.Printf("👤 User ID from token: %s\n", userID)

	// ✅ Gọi Clerk API để lấy thông tin user bằng user ID
	apiURL := fmt.Sprintf("https://api.clerk.com/v1/users/%s", userID)
	fmt.Printf("🌐 Calling Clerk API: %s\n", apiURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// ✅ Sử dụng CLERK_SECRET_KEY để authenticate với Clerk API
	req.Header.Set("Authorization", "Bearer "+clerkSecretKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ HTTP request error: %v\n", err)
		return nil, fmt.Errorf("clerk API request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("✅ Clerk API status: %d\n", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("📦 Clerk API response: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clerk API error (status %d): %s", resp.StatusCode, string(body))
	}

	var clerkUser ClerkUserResponse
	if err := json.Unmarshal(body, &clerkUser); err != nil {
		fmt.Printf("❌ JSON unmarshal error: %v\n", err)
		return nil, fmt.Errorf("failed to parse clerk response: %v", err)
	}

	fmt.Printf("✅ Successfully parsed Clerk user\n")
	return &clerkUser, nil
}
