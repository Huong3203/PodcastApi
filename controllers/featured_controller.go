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

	// ⭐ Lấy top 5 podcast có điểm trung bình cao nhất
	if err := db.Table("podcasts p").
		Select(`
			p.*,
			COALESCE(AVG(d.sao), 0) AS avg_rating,
			COUNT(d.id) AS total_votes
		`).
		Joins("LEFT JOIN danh_gias d ON d.podcast_id = p.id").
		Where("p.trang_thai = ?", "Bật").
		Group("p.id").
		Order("avg_rating DESC, total_votes DESC").
		Limit(5).
		Scan(&podcasts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy podcast nổi bật"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"featured_podcasts": podcasts,
	})
}

// ========================= ĐÁNH GIÁ NỔI BẬT ========================= //

func GetFeaturedRatings(c *gin.Context) {
	db := config.DB

	type RatingWithUser struct {
		models.DanhGia
		UserName string `json:"user_name"`
		Avatar   string `json:"avatar"`
	}

	var ratings []RatingWithUser

	// ⭐ Lấy 10 đánh giá nổi bật (5 sao hoặc mới nhất)
	if err := db.Table("danh_gias d").
		Select(`
			d.*,
			u.ho_ten AS user_name,
			u.avatar AS avatar
		`).
		Joins("LEFT JOIN nguoi_dungs u ON u.id = d.user_id").
		Order("d.sao DESC, d.ngay_tao DESC").
		Limit(10).
		Scan(&ratings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy đánh giá nổi bật"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"featured_ratings": ratings,
	})
}
