package middleware

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/clerkinc/clerk-sdk-go/clerk"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var ClerkClient clerk.Client

func init() {
	secret := os.Getenv("CLERK_SECRET_KEY")
	if secret == "" {
		panic("Thiếu CLERK_SECRET_KEY")
	}

	var err error
	ClerkClient, err = clerk.NewClient(secret)
	if err != nil {
		panic("Không thể khởi tạo Clerk client: " + err.Error())
	}
}

// AuthMiddleware: bắt buộc phải có token (JWT local hoặc Clerk)
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.GetHeader("X-Auth-Token")
		}
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu token"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không đúng định dạng"})
			c.Abort()
			return
		}
		token := parts[1]

		// 1️⃣ JWT local
		if claims, err := utils.VerifyToken(token); err == nil {
			c.Set("user_id", claims.UserID)
			c.Set("role", claims.Role)
			c.Set("provider", "local")

			var user models.NguoiDung
			if err := config.DB.First(&user, "id = ?", claims.UserID).Error; err == nil {
				if !user.KichHoat {
					c.JSON(http.StatusForbidden, gin.H{"error": "Tài khoản đã bị tạm khóa"})
					c.Abort()
					return
				}
			}
			c.Next()
			return
		}

		// 2️⃣ Fallback Clerk
		session, err := ClerkClient.Sessions().Read(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
			c.Abort()
			return
		}

		clerkID := session.UserID
		clerkUser, err := ClerkClient.Users().Read(clerkID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Không lấy được user từ Clerk"})
			c.Abort()
			return
		}

		email := ""
		if len(clerkUser.EmailAddresses) > 0 {
			email = clerkUser.EmailAddresses[0].EmailAddress
		}

		var user models.NguoiDung
		err = config.DB.First(&user, "id = ?", clerkID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = models.NguoiDung{
				ID:       clerkUser.ID,
				Email:    email,
				VaiTro:   "user",
				Provider: "clerk",
				KichHoat: true,
			}
			config.DB.Create(&user)
		}

		c.Set("user_id", user.ID)
		c.Set("email", user.Email)
		c.Set("role", user.VaiTro)
		c.Set("provider", user.Provider)
		c.Next()
	}
}

// OptionalAuthMiddleware: token không bắt buộc
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.GetHeader("X-Auth-Token")
		}
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.Next()
			return
		}
		token := parts[1]

		if claims, err := utils.VerifyToken(token); err == nil {
			c.Set("user_id", claims.UserID)
			c.Set("role", claims.Role)
			c.Set("provider", "local")
			c.Next()
			return
		}

		session, err := ClerkClient.Sessions().Read(token)
		if err != nil {
			c.Next()
			return
		}

		clerkID := session.UserID
		clerkUser, err := ClerkClient.Users().Read(clerkID)
		if err != nil {
			c.Next()
			return
		}

		email := ""
		if len(clerkUser.EmailAddresses) > 0 {
			email = clerkUser.EmailAddresses[0].EmailAddress
		}

		var user models.NguoiDung
		err = config.DB.First(&user, "id = ?", clerkID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = models.NguoiDung{
				ID:       clerkUser.ID,
				Email:    email,
				VaiTro:   "user",
				Provider: "clerk",
				KichHoat: true,
			}
			config.DB.Create(&user)
		}

		c.Set("user_id", user.ID)
		c.Set("email", user.Email)
		c.Set("role", user.VaiTro)
		c.Set("provider", user.Provider)
		c.Next()
	}
}
