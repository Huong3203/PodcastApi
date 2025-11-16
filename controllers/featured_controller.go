package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
)

// GetFeaturedPodcasts lấy top N podcast nổi bật
func GetFeaturedPodcasts(c *gin.Context) {
	var podcasts []models.Podcast
	limit := 10 // giới hạn 10 podcast nổi bật, có thể lấy từ query param

	if err := config.DB.Preload("TaiLieu").Preload("DanhMuc").
		Where("trang_thai = ?", "Bật").
		Order("luot_yeu_thich DESC, luot_xem DESC").
		Limit(limit).Find(&podcasts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy podcast nổi bật"})
		return
	}

	// Lấy tóm tắt từ TaiLieu
	for i := range podcasts {
		podcasts[i].TomTat = podcasts[i].TaiLieu.TomTat
	}

	c.JSON(http.StatusOK, gin.H{"data": podcasts})
}

// GetFeaturedReviews lấy danh sách đánh giá nổi bật (ví dụ: 5 sao gần đây)
func GetFeaturedReviews(c *gin.Context) {
	var danhgias []models.DanhGia
	limit := 10

	if err := config.DB.Preload("Podcast").Preload("User").
		Where("sao = ?", 5).
		Order("ngay_tao DESC").
		Limit(limit).Find(&danhgias).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy đánh giá nổi bật"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": danhgias})
}
