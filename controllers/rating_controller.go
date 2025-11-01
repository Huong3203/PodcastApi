package controllers

import (
	"net/http"
	"strconv"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ==========================
// 🔹 Thêm đánh giá cho podcast
// ==========================
func AddPodcastRating(c *gin.Context) {
	db := config.DB
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn phải đăng nhập"})
		return
	}

	podcastID := c.Param("id")
	saoStr := c.PostForm("sao")
	binhLuan := c.PostForm("binh_luan")

	sao, err := strconv.Atoi(saoStr)
	if err != nil || sao < 1 || sao > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số sao phải là số từ 1 đến 5"})
		return
	}

	rating := models.DanhGia{
		ID:        uuid.New().String(),
		PodcastID: podcastID,
		UserID:    userID,
		Sao:       sao,
		BinhLuan:  binhLuan,
	}

	if err := db.Create(&rating).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm đánh giá"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Đánh giá thành công",
		"rating":  rating,
	})
}

// ==========================
// 🔹 Lấy tất cả đánh giá của podcast
// ==========================
func GetPodcastRatings(c *gin.Context) {
	db := config.DB
	podcastID := c.Param("id")

	var ratings []models.DanhGia
	if err := db.Preload("User").Preload("Podcast").Where("podcast_id = ?", podcastID).Find(&ratings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy đánh giá"})
		return
	}

	// Tính điểm trung bình
	var avg float64
	if err := db.Model(&models.DanhGia{}).Where("podcast_id = ?", podcastID).Select("AVG(sao)").Scan(&avg).Error; err != nil {
		avg = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"ratings":     ratings,
		"avg_rating":  avg,
		"total_votes": len(ratings),
	})
}

// ==========================
// 🔹 Thống kê đánh giá cho admin
// ==========================
func GetAdminRatingsStats(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền truy cập"})
		return
	}

	db := config.DB

	var totalRatings int64
	var avgRating float64

	db.Model(&models.DanhGia{}).Count(&totalRatings)
	db.Model(&models.DanhGia{}).Select("AVG(sao)").Scan(&avgRating)

	c.JSON(http.StatusOK, gin.H{
		"total_ratings": totalRatings,
		"avg_rating":    avgRating,
	})
}
