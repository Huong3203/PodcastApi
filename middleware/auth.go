// package middleware

// import (
// 	"net/http"
// 	"strings"

// 	"github.com/Huong3203/APIPodcast/config"
// 	"github.com/Huong3203/APIPodcast/models"
// 	"github.com/Huong3203/APIPodcast/utils"
// 	"github.com/gin-gonic/gin"
// )

// // AuthMiddleware: bắt buộc phải có JWT token local
// func AuthMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		authHeader := c.GetHeader("Authorization")
// 		if authHeader == "" {
// 			authHeader = c.GetHeader("X-Auth-Token")
// 		}
// 		if authHeader == "" {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu token"})
// 			c.Abort()
// 			return
// 		}

// 		parts := strings.Split(authHeader, " ")
// 		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không đúng định dạng"})
// 			c.Abort()
// 			return
// 		}
// 		token := parts[1]

// 		// ✅ Verify JWT local token
// 		claims, err := utils.VerifyToken(token)
// 		if err != nil {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ hoặc đã hết hạn"})
// 			c.Abort()
// 			return
// 		}

// 		// ✅ Kiểm tra user có bị khoá không
// 		var user models.NguoiDung
// 		if err := config.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Người dùng không tồn tại"})
// 			c.Abort()
// 			return
// 		}

// 		if !user.KichHoat {
// 			c.JSON(http.StatusForbidden, gin.H{"error": "Tài khoản đã bị tạm khóa"})
// 			c.Abort()
// 			return
// 		}

// 		// ✅ Set user info vào context
// 		c.Set("user_id", claims.UserID)
// 		c.Set("role", claims.Role)
// 		c.Set("email", user.Email)
// 		c.Set("provider", user.Provider)
// 		c.Next()
// 	}
// }

// // OptionalAuthMiddleware: token không bắt buộc
// func OptionalAuthMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		authHeader := c.GetHeader("Authorization")
// 		if authHeader == "" {
// 			authHeader = c.GetHeader("X-Auth-Token")
// 		}
// 		if authHeader == "" {
// 			c.Next()
// 			return
// 		}

// 		parts := strings.Split(authHeader, " ")
// 		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
// 			c.Next()
// 			return
// 		}
// 		token := parts[1]

// 		// ✅ Verify JWT local token
// 		claims, err := utils.VerifyToken(token)
// 		if err != nil {
// 			c.Next()
// 			return
// 		}

// 		// ✅ Lấy thông tin user từ DB
// 		var user models.NguoiDung
// 		if err := config.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
// 			c.Next()
// 			return
// 		}

//			// ✅ Set user info vào context
//			c.Set("user_id", claims.UserID)
//			c.Set("role", claims.Role)
//			c.Set("email", user.Email)
//			c.Set("provider", user.Provider)
//			c.Next()
//		}
//	}
package middleware

import (
	"net/http"
	"strings"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware: bắt buộc phải có JWT token local
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

		// ✅ Verify JWT local token
		claims, err := utils.VerifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ hoặc đã hết hạn"})
			c.Abort()
			return
		}

		// ✅ Kiểm tra user có bị khoá không
		var user models.NguoiDung
		if err := config.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Người dùng không tồn tại"})
			c.Abort()
			return
		}

		if !user.KichHoat {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tài khoản đã bị tạm khóa"})
			c.Abort()
			return
		}

		// ✅ Set user info vào context với CẢ HAI KEY để tương thích
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)    // Key mới (tiêu chuẩn)
		c.Set("vai_tro", claims.Role) // Key cũ (để tương thích với code hiện tại)
		c.Set("email", user.Email)
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

		// ✅ Verify JWT local token
		claims, err := utils.VerifyToken(token)
		if err != nil {
			c.Next()
			return
		}

		// ✅ Lấy thông tin user từ DB
		var user models.NguoiDung
		if err := config.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.Next()
			return
		}

		// ✅ Set user info vào context với CẢ HAI KEY
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)    // Key mới
		c.Set("vai_tro", claims.Role) // Key cũ (tương thích)
		c.Set("email", user.Email)
		c.Set("provider", user.Provider)
		c.Next()
	}
}
