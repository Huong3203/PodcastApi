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
		Select("podcasts.*, COALESCE(AVG(danh_gias.sao),0) AS avg_rating, COUNT(danh_gias.id) AS total_votes").
		Joins("LEFT JOIN danh_gias ON danh_gias.podcast_id = podcasts.id").
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

	if err := db.Preload("User").
		Order("sao DESC, ngay_tao DESC").
		Limit(10).
		Find(&ratings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy đánh giá nổi bật", "detail": err.Error()})
		return
	}

	type RatingWithUser struct {
		models.DanhGia
		UserName string `json:"user_name"`
		Avatar   string `json:"avatar"`
	}

	var result []RatingWithUser
	for _, r := range ratings {
		result = append(result, RatingWithUser{
			DanhGia:  r,
			UserName: r.User.HoTen,
			Avatar:   r.User.Avatar,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"featured_ratings": result,
	})
}
