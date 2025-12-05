package controllers

import (
	"net/http"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/ws"
	"github.com/gin-gonic/gin"
)

// ============================================
// USER NOTIFICATIONS
// ============================================

// GetNotifications - Lấy danh sách thông báo của user
func GetNotifications(c *gin.Context) {
	userID := c.GetString("user_id")

	var notifications []models.Notification
	if err := config.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông báo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         len(notifications),
	})
}

// GetUnreadCount - Đếm số thông báo chưa đọc
func GetUnreadCount(c *gin.Context) {
	userID := c.GetString("user_id")

	var count int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

// MarkNotificationAsRead - Đánh dấu 1 thông báo đã đọc
func MarkNotificationAsRead(c *gin.Context) {
	userID := c.GetString("user_id")
	notificationID := c.Param("id")

	now := time.Now()
	result := config.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông báo"})
		return
	}

	// Cập nhật badge realtime
	var count int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count)
	ws.SendBadgeUpdate(userID, count)

	c.JSON(http.StatusOK, gin.H{"message": "Đã đánh dấu đã đọc"})
}

// MarkAllAsRead - Đánh dấu tất cả thông báo đã đọc
func MarkAllAsRead(c *gin.Context) {
	userID := c.GetString("user_id")

	now := time.Now()
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		})

	// Badge = 0
	ws.SendBadgeUpdate(userID, 0)

	c.JSON(http.StatusOK, gin.H{"message": "Đã đánh dấu tất cả đã đọc"})
}

// DeleteNotification - Xóa 1 thông báo
func DeleteNotification(c *gin.Context) {
	userID := c.GetString("user_id")
	notificationID := c.Param("id")

	result := config.DB.Delete(&models.Notification{}, "id = ? AND user_id = ?", notificationID, userID)

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông báo"})
		return
	}

	// Cập nhật badge
	var count int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count)
	ws.SendBadgeUpdate(userID, count)

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thông báo"})
}

// DeleteAllNotifications - Xóa tất cả thông báo
func DeleteAllNotifications(c *gin.Context) {
	userID := c.GetString("user_id")

	config.DB.Delete(&models.Notification{}, "user_id = ?", userID)

	// Badge = 0
	ws.SendBadgeUpdate(userID, 0)

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa tất cả thông báo"})
}

// DeleteReadNotifications - Xóa các thông báo đã đọc
func DeleteReadNotifications(c *gin.Context) {
	userID := c.GetString("user_id")

	config.DB.Delete(&models.Notification{}, "user_id = ? AND is_read = true", userID)

	// Cập nhật badge (chỉ còn unread)
	var count int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count)
	ws.SendBadgeUpdate(userID, count)

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thông báo đã đọc"})
}

// ============================================
// ADMIN NOTIFICATIONS
// ============================================

// GetAdminNotifications - Lấy danh sách thông báo admin
func GetAdminNotifications(c *gin.Context) {
	adminID := c.GetString("user_id")

	var notifications []models.AdminNotification
	if err := config.DB.
		Where("admin_id = ?", adminID).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông báo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         len(notifications),
	})
}

// GetAdminUnreadCount - Đếm số thông báo admin chưa đọc
func GetAdminUnreadCount(c *gin.Context) {
	adminID := c.GetString("user_id")

	var count int64
	config.DB.Model(&models.AdminNotification{}).
		Where("admin_id = ? AND is_read = false", adminID).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

// MarkAdminNotificationAsRead - Đánh dấu 1 thông báo admin đã đọc
func MarkAdminNotificationAsRead(c *gin.Context) {
	adminID := c.GetString("user_id")
	notificationID := c.Param("id")

	now := time.Now()
	result := config.DB.Model(&models.AdminNotification{}).
		Where("id = ? AND admin_id = ?", notificationID, adminID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông báo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã đánh dấu đã đọc"})
}

// MarkAllAdminAsRead - Đánh dấu tất cả thông báo admin đã đọc
func MarkAllAdminAsRead(c *gin.Context) {
	adminID := c.GetString("user_id")

	now := time.Now()
	config.DB.Model(&models.AdminNotification{}).
		Where("admin_id = ? AND is_read = false", adminID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		})

	c.JSON(http.StatusOK, gin.H{"message": "Đã đánh dấu tất cả đã đọc"})
}

// DeleteAdminNotification - Xóa 1 thông báo admin
func DeleteAdminNotification(c *gin.Context) {
	adminID := c.GetString("user_id")
	notificationID := c.Param("id")

	result := config.DB.Delete(&models.AdminNotification{}, "id = ? AND admin_id = ?", notificationID, adminID)

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông báo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thông báo"})
}

// DeleteAllAdminNotifications - Xóa tất cả thông báo admin
func DeleteAllAdminNotifications(c *gin.Context) {
	adminID := c.GetString("user_id")

	config.DB.Delete(&models.AdminNotification{}, "admin_id = ?", adminID)

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa tất cả thông báo"})
}

// DeleteReadAdminNotifications - Xóa các thông báo admin đã đọc
func DeleteReadAdminNotifications(c *gin.Context) {
	adminID := c.GetString("user_id")

	config.DB.Delete(&models.AdminNotification{}, "admin_id = ? AND is_read = true", adminID)

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thông báo đã đọc"})
}
