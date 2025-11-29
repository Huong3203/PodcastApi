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

// AuthMiddleware xác thực JWT local hoặc token Clerk
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
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không đúng định dạng"})
			c.Abort()
			return
		}
		token := parts[1]

		// 1️⃣ Kiểm tra JWT local
		if claims, err := utils.VerifyToken(token); err == nil {
			c.Set("user_id", claims.UserID)
			c.Set("vai_tro", claims.Role)
			c.Set("provider", "local")
			c.Next()
			return
		}

		// 2️⃣ Kiểm tra token Clerk
		sess, err := ClerkClient.Sessions().VerifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
			c.Abort()
			return
		}

		clerkID := sess.Subject // Đây mới là userID đúng của Clerk

		// 3️⃣ Lấy thông tin user từ Clerk
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

		// 4️⃣ Lưu hoặc lấy user trong DB
		var user models.NguoiDung
		err = config.DB.Where("id = ?", clerkID).First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Chưa có user → tạo mới
			user = models.NguoiDung{
				ID:       clerkID,
				Email:    email,
				Provider: "clerk",
				VaiTro:   "user",
				KichHoat: true,
			}
			config.DB.Create(&user)
		} else if err != nil {
			// Lỗi DB khác
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
			c.Abort()
			return
		}

		// 5️⃣ Set context
		c.Set("user_id", user.ID)
		c.Set("email", user.Email)
		c.Set("vai_tro", user.VaiTro)
		c.Set("provider", "clerk")

		c.Next()
	}
}
