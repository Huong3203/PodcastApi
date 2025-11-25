package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"

	"github.com/clerkinc/clerk-sdk-go/clerk"
)

var clerkClient clerk.Client

// INIT CHỈ TẠO 1 LẦN
func init() {
	var err error
	clerkClient, err = clerk.NewClient()
	if err != nil {
		panic("Không thể tạo Clerk client: " + err.Error())
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Lấy token từ header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.GetHeader("X-Auth-Token")
		}

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu Authorization header"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header không hợp lệ"})
			c.Abort()
			return
		}

		token := parts[1]

		// =====================================
		// 1. Kiểm tra JWT local (ưu tiên)
		// =====================================
		claims, err := utils.VerifyToken(token)
		if err == nil {
			c.Set("user_id", claims.UserID)
			c.Set("vai_tro", claims.Role)
			c.Set("provider", "local")
			c.Next()
			return
		}

		// =====================================
		// 2. Kiểm tra token Clerk
		// =====================================
		sess, err := clerkClient.VerifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ (local + Clerk đều fail)"})
			c.Abort()
			return
		}

		clerkID := sess.Subject

		// =====================================
		// 3. Lấy thông tin user từ Clerk API
		// =====================================
		clerkUser, err := clerkClient.Users().Read(clerkID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Không thể đọc user từ Clerk"})
			c.Abort()
			return
		}

		// Lấy email
		email := ""
		if len(clerkUser.EmailAddresses) > 0 {
			email = clerkUser.EmailAddresses[0].EmailAddress
		}

		// =====================================
		// 4. Lưu hoặc lấy user từ database
		// =====================================
		var user models.NguoiDung
		result := config.DB.Where("id = ?", clerkID).First(&user)

		if result.Error != nil {
			user = models.NguoiDung{
				ID:       clerkID,
				Email:    email,
				Provider: "clerk",
				VaiTro:   "user",
				KichHoat: true,
			}
			config.DB.Create(&user)
		}

		// Gửi vào context để controller dùng
		c.Set("user_id", user.ID)
		c.Set("email", user.Email)
		c.Set("vai_tro", user.VaiTro)
		c.Set("provider", "clerk")

		c.Next()
	}
}
