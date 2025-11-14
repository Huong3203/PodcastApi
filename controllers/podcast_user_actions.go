package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LƯU LỊCH SỬ NGHE
func LuuLichSuNghe(c *gin.Context) {
	var body struct {
		PodcastID string `json:"podcast_id"`
		ViTri     int    `json:"vi_tri"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn cần đăng nhập"})
		return
	}

	var history models.LichSuNghe
	err := config.DB.Where("nguoi_dung_id = ? AND podcast_id = ?", userID, body.PodcastID).
		First(&history).Error

	if err == gorm.ErrRecordNotFound {
		history = models.LichSuNghe{
			ID:          uuid.New().String(),
			NguoiDungID: userID,
			PodcastID:   body.PodcastID,
			ViTri:       body.ViTri,
			NgayNghe:    time.Now(),
		}
		config.DB.Create(&history)
	} else {
		config.DB.Model(&history).Updates(models.LichSuNghe{
			ViTri:    body.ViTri,
			NgayNghe: time.Now(),
		})
	}

	// 🔹 Tạo thông báo (nếu muốn ghi lại lịch sử nghe)
	message := fmt.Sprintf("Người dùng đã nghe podcast %s", body.PodcastID)
	if err := services.CreateNotification(userID, body.PodcastID, "listened", message); err != nil {
		fmt.Println("Lỗi khi tạo thông báo:", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã lưu lịch sử nghe"})
}

// YÊU THÍCH PODCAST
func ToggleYeuThichPodcast(c *gin.Context) {
	podcastID := c.Param("id")
	userID := c.GetString("user_id")

	var fav models.PodcastYeuThich
	err := config.DB.Where("nguoi_dung_id = ? AND podcast_id = ?", userID, podcastID).First(&fav).Error

	if err == gorm.ErrRecordNotFound {
		// Thêm yêu thích
		fav = models.PodcastYeuThich{
			ID:          uuid.New().String(),
			NguoiDungID: userID,
			PodcastID:   podcastID,
		}
		config.DB.Create(&fav)
		config.DB.Model(&models.Podcast{}).Where("id = ?", podcastID).
			UpdateColumn("luot_yeu_thich", gorm.Expr("luot_yeu_thich + 1"))

		// 🔹 Tạo thông báo realtime
		message := fmt.Sprintf("Người dùng %s đã yêu thích podcast %s", userID, podcastID)
		services.CreateNotification(userID, podcastID, "favorite", message)

		c.JSON(http.StatusOK, gin.H{"message": "Đã yêu thích"})
		return
	}

	// Bỏ yêu thích
	config.DB.Delete(&fav)
	config.DB.Model(&models.Podcast{}).Where("id = ?", podcastID).
		UpdateColumn("luot_yeu_thich", gorm.Expr("luot_yeu_thich - 1"))

	// 🔹 Tạo thông báo realtime
	message := fmt.Sprintf("Người dùng %s đã bỏ yêu thích podcast %s", userID, podcastID)
	services.CreateNotification(userID, podcastID, "unfavorite", message)

	c.JSON(http.StatusOK, gin.H{"message": "Đã bỏ yêu thích"})
}

// LƯU PODCAST VÀO THƯ VIỆN
func ToggleLuuPodcast(c *gin.Context) {
	podcastID := c.Param("id")
	userID := c.GetString("user_id")

	var save models.PodcastLuu
	err := config.DB.Where("nguoi_dung_id = ? AND podcast_id = ?", userID, podcastID).First(&save).Error

	if err == gorm.ErrRecordNotFound {
		// Lưu podcast
		save = models.PodcastLuu{
			ID:          uuid.New().String(),
			NguoiDungID: userID,
			PodcastID:   podcastID,
		}
		config.DB.Create(&save)
		config.DB.Model(&models.Podcast{}).Where("id = ?", podcastID).
			UpdateColumn("luot_luu", gorm.Expr("luot_luu + 1"))

		// 🔹 Tạo thông báo realtime
		message := fmt.Sprintf("Người dùng %s đã lưu podcast %s vào thư viện", userID, podcastID)
		services.CreateNotification(userID, podcastID, "saved", message)

		c.JSON(http.StatusOK, gin.H{"message": "Đã lưu podcast"})
		return
	}

	// Bỏ lưu podcast
	config.DB.Delete(&save)
	config.DB.Model(&models.Podcast{}).Where("id = ?", podcastID).
		UpdateColumn("luot_luu", gorm.Expr("luot_luu - 1"))

	// 🔹 Tạo thông báo realtime
	message := fmt.Sprintf("Người dùng %s đã bỏ lưu podcast %s", userID, podcastID)
	services.CreateNotification(userID, podcastID, "unsaved", message)

	c.JSON(http.StatusOK, gin.H{"message": "Đã bỏ lưu"})
}

// LẤY DANH SÁCH YÊU THÍCH CỦA NGƯỜI DÙNG
func GetMyFavoritePodcasts(c *gin.Context) {
	userID := c.GetString("user_id")

	var list []models.PodcastYeuThich
	config.DB.Preload("Podcast").Preload("Podcast.TaiLieu").Where("nguoi_dung_id = ?", userID).
		Order("ngay_thich DESC").Find(&list)

	var result []models.Podcast
	for _, item := range list {
		// Gán TomTat ra root để frontend không cần nested
		item.Podcast.TomTat = item.Podcast.TaiLieu.TomTat
		result = append(result, item.Podcast)
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// LẤY DANH SÁCH ĐÃ LƯU
func GetMySavedPodcasts(c *gin.Context) {
	userID := c.GetString("user_id")

	var list []models.PodcastLuu
	config.DB.Preload("Podcast").Preload("Podcast.TaiLieu").Where("nguoi_dung_id = ?", userID).
		Order("ngay_luu DESC").Find(&list)

	var result []models.Podcast
	for _, s := range list {
		s.Podcast.TomTat = s.Podcast.TaiLieu.TomTat
		result = append(result, s.Podcast)
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// LẤY LỊCH SỬ NGHE
func GetMyListeningHistory(c *gin.Context) {
	userID := c.GetString("user_id")

	var list []models.LichSuNghe
	config.DB.Preload("Podcast").Preload("Podcast.TaiLieu").Where("nguoi_dung_id = ?", userID).
		Order("ngay_nghe DESC").Find(&list)

	// Gán TomTat
	for i := range list {
		list[i].Podcast.TomTat = list[i].Podcast.TaiLieu.TomTat
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}
