package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ToggleLuuPodcast(c *gin.Context) {
	podcastID := c.Param("id")
	userID := c.GetString("user_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn phải đăng nhập"})
		return
	}

	var save models.PodcastLuu

	err := config.DB.Where("nguoi_dung_id = ? AND podcast_id = ?", userID, podcastID).First(&save).Error

	// Nếu chưa lưu → lưu
	if err == gorm.ErrRecordNotFound {
		save = models.PodcastLuu{
			ID:          uuid.New().String(),
			NguoiDungID: userID,
			PodcastID:   podcastID,
		}
		config.DB.Create(&save)

		config.DB.Model(&models.Podcast{}).Where("id = ?", podcastID).UpdateColumn("luot_luu", gorm.Expr("luot_luu + 1"))

		c.JSON(http.StatusOK, gin.H{"message": "Đã lưu podcast"})
		return
	}

	// Nếu đã lưu → bỏ lưu
	config.DB.Delete(&save)
	config.DB.Model(&models.Podcast{}).Where("id = ?", podcastID).UpdateColumn("luot_luu", gorm.Expr("luot_luu - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "Đã bỏ lưu"})
}
