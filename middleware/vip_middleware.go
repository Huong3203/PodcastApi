package middleware

import (
	"net/http"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
)

// ✅ Middleware kiểm tra VIP (chặn nếu không có VIP) - HARD BLOCK
func RequireVIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")

		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Vui lòng đăng nhập để tiếp tục",
			})
			c.Abort()
			return
		}

		db := config.DB
		var user models.NguoiDung
		if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Server Error",
				"message": "Không thể xác thực người dùng",
			})
			c.Abort()
			return
		}

		// Kiểm tra VIP
		if !user.VIP {
			c.JSON(http.StatusForbidden, gin.H{
				"error":        "VIP Required",
				"message":      "Tính năng này chỉ dành cho thành viên VIP",
				"requires_vip": true,
			})
			c.Abort()
			return
		}

		// Kiểm tra VIP còn hạn
		if user.VIPExpires != nil && time.Now().After(*user.VIPExpires) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":        "VIP Expired",
				"message":      "VIP của bạn đã hết hạn. Vui lòng gia hạn để tiếp tục sử dụng",
				"vip_expired":  true,
				"requires_vip": true,
			})
			c.Abort()
			return
		}

		// Set flag để biết user có VIP
		c.Set("is_vip", true)
		c.Set("vip_expires", user.VIPExpires)
		c.Set("user_vip_info", gin.H{
			"has_vip":    true,
			"expires_at": user.VIPExpires,
			"auto_renew": user.AutoRenew,
		})
		c.Next()
	}
}

// ✅ Middleware kiểm tra VIP nhẹ (chỉ set flag, không chặn) - SOFT CHECK
func CheckVIPStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")

		isVIP := false
		var vipExpires *time.Time
		var autoRenew bool

		if userIDStr != "" {
			db := config.DB
			var user models.NguoiDung
			if err := db.First(&user, "id = ?", userIDStr).Error; err == nil {
				if user.VIP {
					if user.VIPExpires == nil || time.Now().Before(*user.VIPExpires) {
						isVIP = true
						vipExpires = user.VIPExpires
						autoRenew = user.AutoRenew
					}
				}
			}
		}

		c.Set("is_vip", isVIP)
		c.Set("vip_expires", vipExpires)
		c.Set("vip_auto_renew", autoRenew)
		c.Next()
	}
}

// ✅ Middleware tự động kiểm tra gia hạn VIP nếu có auto_renew
func AutoRenewVIPCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")

		if userIDStr != "" {
			db := config.DB
			var user models.NguoiDung
			if err := db.First(&user, "id = ?", userIDStr).Error; err == nil {
				// Kiểm tra nếu VIP hết hạn và có auto_renew
				if user.VIP && user.AutoRenew && user.VIPExpires != nil {
					now := time.Now()

					// Nếu VIP đã hết hạn
					if now.After(*user.VIPExpires) {
						c.Set("vip_renewal_needed", true)
						c.Set("vip_expired_at", user.VIPExpires)

						// TODO: Tích hợp với payment gateway để tự động gia hạn
						// services.AutoRenewVIPSubscription(user.ID)
					} else if now.Add(3 * 24 * time.Hour).After(*user.VIPExpires) {
						// Nếu VIP sắp hết hạn trong 3 ngày
						c.Set("vip_renewal_reminder", true)
						c.Set("vip_expires_soon", user.VIPExpires)
					}
				}
			}
		}

		c.Next()
	}
}

// ✅ Middleware kiểm tra quyền admin hoặc VIP
func RequireAdminOrVIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")
		role, _ := c.Get("vai_tro")

		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Vui lòng đăng nhập",
			})
			c.Abort()
			return
		}

		// Nếu là admin thì pass
		if role == "admin" {
			c.Set("is_admin", true)
			c.Next()
			return
		}

		// Nếu không phải admin, kiểm tra VIP
		db := config.DB
		var user models.NguoiDung
		if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể xác thực người dùng",
			})
			c.Abort()
			return
		}

		// Kiểm tra VIP
		if !user.VIP || (user.VIPExpires != nil && time.Now().After(*user.VIPExpires)) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":        "Access Denied",
				"message":      "Tính năng này yêu cầu quyền Admin hoặc VIP",
				"requires_vip": true,
			})
			c.Abort()
			return
		}

		c.Set("is_vip", true)
		c.Next()
	}
}
