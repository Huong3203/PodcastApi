package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"

	"github.com/clerkinc/clerk-sdk-go/clerk"
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

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.GetHeader("X-Auth-Token")
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu token"})
			c.Abort()
			return
		}

		parts := strings.Split(token, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không đúng định dạng"})
			c.Abort()
			return
		}

		token = parts[1]

		// 1. ---------------------------
		// KIỂM TRA JWT LOCAL TRƯỚC
		// ---------------------------
		if claims, err := utils.VerifyToken(token); err == nil {
			c.Set("user_id", claims.UserID)
			c.Set("vai_tro", claims.Role)
			c.Set("provider", "local")
			c.Next()
			return
		}

		// 2. ---------------------------
		// FALLBACK: XÁC THỰC TOKEN CLERK
		// ---------------------------
		sess, err := ClerkClient.Sessions().VerifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
			c.Abort()
			return
		}

		clerkID := sess.UserID

		// 3. Lấy User Clerk
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

		// 4. Lưu user vào DB nếu chưa có
		var user models.NguoiDung
		err = config.DB.Where("id = ?", clerkID).First(&user).Error

		if err != nil {
			user = models.NguoiDung{
				ID:       clerkID,
				Email:    email,
				Provider: "clerk",
				VaiTro:   "user",
				KichHoat: true,
			}
			config.DB.Create(&user)
		}

		// Set vào context
		c.Set("user_id", user.ID)
		c.Set("email", user.Email)
		c.Set("vai_tro", user.VaiTro)
		c.Set("provider", "clerk")

		c.Next()
	}
}
