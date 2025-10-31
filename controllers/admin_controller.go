package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
)

func GetAdminStats(c *gin.Context) {
	db := config.DB

	var userCount int64
	var podcastCount int64
	var totalViews int64
	// Kiểm tra quyền truy cập, chỉ admin mới có thể xem thống kê
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền truy cập"})
		return
	}

	// Đếm tổng số người dùng
	if err := db.Model(&models.NguoiDung{}).Count(&userCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm số lượng người dùng"})
		return
	}

	// Đếm tổng số podcast
	if err := db.Model(&models.Podcast{}).Count(&podcastCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm số podcast"})
		return
	}

	// Tính tổng lượt xem
	if err := db.Model(&models.Podcast{}).Select("SUM(luot_xem)").Scan(&totalViews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tính tổng lượt xem"})
		return
	}

	// Đếm trạng thái tài liệu
	var processingCount, completedCount int64
	db.Model(&models.TaiLieu{}).Where("trang_thai = ?", "Đã tải lên").Count(&processingCount)
	db.Model(&models.TaiLieu{}).Where("trang_thai = ?", "Hoàn thành").Count(&completedCount)

	c.JSON(http.StatusOK, gin.H{
		"total_users":          userCount,
		"total_podcasts":       podcastCount,
		"total_views":          totalViews,
		"documents_processing": processingCount,
		"documents_done":       completedCount,
	})
}
