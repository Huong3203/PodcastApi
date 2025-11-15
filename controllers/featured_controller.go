package controllers

import (
	"net/http"
	"strconv"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
)

// ==========================
// 📌 LẤY DANH SÁCH PODCAST NỔI BẬT
// ==========================
func GetFeaturedPodcasts(c *gin.Context) {
	db := config.DB

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var featured []models.FeaturedPodcast
	var total int64

	query := db.Model(&models.FeaturedPodcast{}).
		Preload("Podcast").
		Preload("Podcast.DanhMuc").
		Preload("Podcast.TaiLieu")

	// Đếm tổng số
	query.Count(&total)

	// Lấy dữ liệu
	if err := query.
		Order("featured_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&featured).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy danh sách podcast nổi bật",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": featured,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// 📌 LẤY DANH SÁCH ĐÁNH GIÁ NỔI BẬT
// ==========================
func GetFeaturedRatings(c *gin.Context) {
	db := config.DB

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var featured []models.FeaturedRating
	var total int64

	query := db.Model(&models.FeaturedRating{}).
		Preload("User").
		Preload("Podcast").
		Preload("Podcast.DanhMuc")

	query.Count(&total)

	if err := query.
		Order("featured_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&featured).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Không thể lấy đánh giá nổi bật",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": featured,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}
