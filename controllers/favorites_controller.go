package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== ADD FAVORITE ====================
func AddFavorite(c *gin.Context) {
	userID := c.GetString("user_id")
	podcastID := c.Param("podcast_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn cần đăng nhập"})
		return
	}

	// Kiểm tra tồn tại chưa
	var fav models.PodcastYeuThich
	err := config.DB.Where("nguoi_dung_id = ? AND podcast_id = ?", userID, podcastID).
		First(&fav).Error

	if err == nil {
		// Đã tồn tại
		c.JSON(http.StatusOK, gin.H{"message": "Đã yêu thích trước đó"})
		return
	}

	// Tạo mới
	fav = models.PodcastYeuThich{
		ID:          uuid.New().String(),
		NguoiDungID: userID,
		PodcastID:   podcastID,
	}

	if err := config.DB.Create(&fav).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thêm yêu thích"})
		return
	}

	// +1 lượt yêu thích
	config.DB.Model(&models.Podcast{}).
		Where("id = ?", podcastID).
		UpdateColumn("luot_yeu_thich", gorm.Expr("luot_yeu_thich + 1"))

	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm yêu thích"})
}

// ==================== REMOVE FAVORITE ====================
func RemoveFavorite(c *gin.Context) {
	userID := c.GetString("user_id")
	podcastID := c.Param("podcast_id")

	var fav models.PodcastYeuThich
	err := config.DB.Where("nguoi_dung_id = ? AND podcast_id = ?", userID, podcastID).
		First(&fav).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Podcast không nằm trong danh sách yêu thích"})
		return
	}

	config.DB.Delete(&fav)

	// -1 lượt yêu thích
	config.DB.Model(&models.Podcast{}).
		Where("id = ?", podcastID).
		UpdateColumn("luot_yeu_thich", gorm.Expr("luot_yeu_thich - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "Đã bỏ yêu thích"})
}

// ==================== CHECK FAVORITE ====================
func CheckFavorite(c *gin.Context) {
	userID := c.GetString("user_id")
	podcastID := c.Param("podcast_id")

	var fav models.PodcastYeuThich
	err := config.DB.Where("nguoi_dung_id = ? AND podcast_id = ?", userID, podcastID).
		First(&fav).Error

	c.JSON(http.StatusOK, gin.H{
		"is_favorite": err == nil,
	})
}

// ==================== GET FAVORITES ====================
func GetFavorites(c *gin.Context) {
	userID := c.GetString("user_id")

	var list []models.PodcastYeuThich
	config.DB.Preload("Podcast").Preload("Podcast.TaiLieu").
		Where("nguoi_dung_id = ?", userID).
		Order("ngay_thich DESC").
		Find(&list)

	// Chuẩn hóa dữ liệu trả về
	var result []gin.H
	for _, f := range list {
		result = append(result, gin.H{
			"id":         f.Podcast.ID,
			"tieu_de":    f.Podcast.TieuDe,
			"mo_ta":      f.Podcast.MoTa,
			"hinh_anh":   f.Podcast.HinhAnhDaiDien,
			"tom_tat":    f.Podcast.TaiLieu.TomTat,
			"ngay_thich": f.NgayThich,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
