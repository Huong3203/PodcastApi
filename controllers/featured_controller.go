package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
)

// ========================= PODCAST NỔI BẬT ========================= //
func GetFeaturedPodcasts(c *gin.Context) {
	db := config.DB

	type PodcastWithStats struct {
		models.Podcast
		AvgRating  float64 `json:"avg_rating"`
		TotalVotes int64   `json:"total_votes"`
	}

	var podcasts []PodcastWithStats

	if err := db.Model(&models.Podcast{}).
		Select("podcasts.*, COALESCE(AVG(danh_gia.sao),0) AS avg_rating, COUNT(danh_gia.id) AS total_votes").
		Joins("LEFT JOIN danh_gia ON danh_gias.podcast_id = podcasts.id").
		Where("podcasts.trang_thai = ?", "Bật").
		Group("podcasts.id").
		Order("avg_rating DESC, total_votes DESC").
		Limit(5).
		Scan(&podcasts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy podcast nổi bật", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"featured_podcasts": podcasts,
	})
}

// ========================= ĐÁNH GIÁ NỔI BẬT ========================= //
func GetFeaturedRatings(c *gin.Context) {
	db := config.DB

	var ratings []models.DanhGia

	// Preload User + Podcast + Podcast.Tailieu + Podcast.Danhmuc
	if err := db.Preload("User").
		Preload("Podcast").
		Preload("Podcast.Tailieu").
		Preload("Podcast.Danhmuc").
		Order("sao DESC, ngay_tao DESC").
		Limit(10).
		Find(&ratings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy đánh giá nổi bật", "detail": err.Error()})
		return
	}

	type RatingWithUserAndPodcast struct {
		models.DanhGia
		UserName string          `json:"user_name"`
		Avatar   string          `json:"avatar"`
		Podcast  *models.Podcast `json:"podcast,omitempty"`
	}

	var result []RatingWithUserAndPodcast
	for _, r := range ratings {
		result = append(result, RatingWithUserAndPodcast{
			DanhGia:  r,
			UserName: r.User.HoTen,
			Avatar:   r.User.Avatar,
			Podcast:  &r.Podcast, // gán luôn podcast
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"featured_ratings": result,
	})
}
