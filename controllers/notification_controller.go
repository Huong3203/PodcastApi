package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
)

// ---------------- USER ----------------

// Lấy tất cả thông báo của user
func GetMyNotifications(c *gin.Context) {
	userID := c.GetString("user_id")

	var notifications []models.Notification
	if err := config.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông báo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": notifications})
}

// Đánh dấu 1 thông báo của user đã đọc
func MarkMyNotificationAsRead(c *gin.Context) {
	userID := c.GetString("user_id")
	notiID := c.Param("id")

	result := config.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notiID, userID).
		Update("is_read", true)

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông báo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã đánh dấu là đã đọc"})
}

// Đánh dấu tất cả thông báo của user đã đọc
func MarkAllMyNotificationsAsRead(c *gin.Context) {
	userID := c.GetString("user_id")

	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true)

	c.JSON(http.StatusOK, gin.H{"message": "Đã đánh dấu tất cả là đã đọc"})
}

// ---------------- ADMIN ----------------

// Lấy tất cả thông báo admin
func GetAllNotifications(c *gin.Context) {
	var notifications []models.Notification
	config.DB.Order("created_at DESC").Find(&notifications)
	c.JSON(http.StatusOK, gin.H{"data": notifications})
}

// Lấy thông báo theo action
func GetNotificationsByAction(c *gin.Context) {
	action := c.Query("action")
	if action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu tham số action"})
		return
	}

	var notifications []models.Notification
	config.DB.Where("action = ?", action).Order("created_at DESC").Find(&notifications)
	c.JSON(http.StatusOK, gin.H{"data": notifications})
}

// Đánh dấu 1 thông báo admin đã đọc
func MarkNotificationAsRead(c *gin.Context) {
	id := c.Param("id")

	result := config.DB.Model(&models.Notification{}).
		Where("id = ?", id).
		Update("is_read", true)

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông báo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã đánh dấu đã đọc"})
}

// Đánh dấu tất cả admin đã đọc
func MarkAllAsRead(c *gin.Context) {
	config.DB.Model(&models.Notification{}).
		Where("is_read = ?", false).
		Update("is_read", true)

	c.JSON(http.StatusOK, gin.H{"message": "Đã đánh dấu tất cả là đã đọc"})
}

// Xóa thông báo
func DeleteNotification(c *gin.Context) {
	id := c.Param("id")
	result := config.DB.Delete(&models.Notification{}, "id = ?", id)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông báo để xóa"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thông báo"})
}
