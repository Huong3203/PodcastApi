package middleware

import (
	"net/http"
	"strings"

	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.GetHeader("X-Auth-Token") // iOS
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

		tokenStr := parts[1]
		claims, err := utils.VerifyToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ hoặc hết hạn"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("vai_tro", claims.Role)
		c.Set("provider", claims.Provider)
		c.Next()
	}
}
