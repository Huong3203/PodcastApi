package controllers

import (
	"net/http"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/ws"
	"github.com/gin-gonic/gin"
)

// Lấy toàn bộ thông báo của user
func GetMyNotifications(c *gin.Context) {
	userID := c.GetString("user_id")

	var list []models.Notification
	if err := config.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông báo"})
		return
	}

	c.JSON(http.StatusOK, list)
}

// Đếm số thông báo chưa đọc
func GetMyUnreadCount(c *gin.Context) {
	userID := c.GetString("user_id")

	var count int64
	config.DB.
		Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

// Đánh dấu 1 thông báo đã đọc
func MarkMyNotificationAsRead(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	now := time.Now()
	result := config.DB.
		Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
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

	c.JSON(http.StatusOK, gin.H{"message": "Đã đọc"})
}

// Đánh dấu tất cả thông báo là đã đọc
func MarkAllMyNotificationsAsRead(c *gin.Context) {
	userID := c.GetString("user_id")

	now := time.Now()
	config.DB.
		Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		})

	ws.SendBadgeUpdate(userID, 0)
	c.JSON(http.StatusOK, gin.H{"message": "Đã đọc tất cả"})
}

// Xóa 1 thông báo
func DeleteMyNotification(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	result := config.DB.Delete(&models.Notification{}, "id = ? AND user_id = ?", id, userID)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông báo"})
		return
	}

	var count int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count)
	ws.SendBadgeUpdate(userID, count)

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa"})
}

// Xóa tất cả thông báo của user
func DeleteAllMyNotifications(c *gin.Context) {
	userID := c.GetString("user_id")

	config.DB.Delete(&models.Notification{}, "user_id = ?", userID)
	ws.SendBadgeUpdate(userID, 0)

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa toàn bộ"})
}

// Xóa các thông báo đã đọc
func DeleteMyReadNotifications(c *gin.Context) {
	userID := c.GetString("user_id")

	config.DB.Delete(&models.Notification{}, "user_id = ? AND is_read = true", userID)

	var count int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count)
	ws.SendBadgeUpdate(userID, count)

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thông báo đã đọc"})
}

func GetAllNotifications(c *gin.Context) {
	var list []models.Notification
	config.DB.Order("created_at DESC").Find(&list)
	c.JSON(http.StatusOK, list)
}

func GetNotificationsByAction(c *gin.Context) {
	action := c.Query("action")
	if action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu action"})
		return
	}

	var list []models.Notification
	config.DB.Where("action = ?", action).
		Order("created_at DESC").
		Find(&list)

	c.JSON(http.StatusOK, list)
}

func MarkNotificationAsRead(c *gin.Context) {
	id := c.Param("id")
	now := time.Now()

	result := config.DB.
		Model(&models.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông báo"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã đọc"})
}

func MarkAllAsRead(c *gin.Context) {
	now := time.Now()

	config.DB.Model(&models.Notification{}).
		Where("is_read = false").
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		})

	c.JSON(http.StatusOK, gin.H{"message": "Đã đọc tất cả"})
}

func DeleteNotification(c *gin.Context) {
	id := c.Param("id")

	result := config.DB.Delete(&models.Notification{}, "id = ?", id)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa"})
}
