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

// ==================== LƯU LỊCH SỬ NGHE ====================
func LuuLichSuNghe(c *gin.Context) {
	var body struct {
		PodcastID string `json:"podcast_id" binding:"required"`
		ViTri     int    `json:"vi_tri" binding:"min=0"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ", "detail": err.Error()})
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
		// Tạo mới lịch sử
		history = models.LichSuNghe{
			ID:          uuid.New().String(),
			NguoiDungID: userID,
			PodcastID:   body.PodcastID,
			ViTri:       body.ViTri,
			NgayNghe:    time.Now(),
		}
		if err := config.DB.Create(&history).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu lịch sử", "detail": err.Error()})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn cơ sở dữ liệu", "detail": err.Error()})
		return
	} else {
		// Cập nhật vị trí và thời gian
		if err := config.DB.Model(&history).Updates(map[string]interface{}{
			"vi_tri":    body.ViTri,
			"ngay_nghe": time.Now(),
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật lịch sử", "detail": err.Error()})
			return
		}
	}

	// Tạo thông báo
	message := fmt.Sprintf("Tiến độ nghe podcast đã được cập nhật đến %d giây", body.ViTri)
	_ = services.CreateNotification(userID, body.PodcastID, "listen_progress", message)

	c.JSON(http.StatusOK, gin.H{"message": "Đã lưu lịch sử nghe", "vi_tri": body.ViTri})
}

// ==================== YÊU THÍCH PODCAST ====================
func ToggleYeuThichPodcast(c *gin.Context) {
	podcastID := c.Param("id")
	if podcastID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu podcast_id"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn cần đăng nhập"})
		return
	}

	var fav models.PodcastYeuThich
	err := config.DB.Where("nguoi_dung_id = ? AND podcast_id = ?", userID, podcastID).First(&fav).Error

	if err == gorm.ErrRecordNotFound {
		// Thêm vào yêu thích
		fav = models.PodcastYeuThich{
			ID:          uuid.New().String(),
			NguoiDungID: userID,
			PodcastID:   podcastID,
		}
		if err := config.DB.Create(&fav).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm yêu thích", "detail": err.Error()})
			return
		}

		// Tăng số lượt yêu thích
		config.DB.Model(&models.Podcast{}).Where("id = ?", podcastID).
			UpdateColumn("luot_yeu_thich", gorm.Expr("luot_yeu_thich + 1"))

		// Thông báo
		_ = services.CreateNotification(userID, podcastID, "favorite_added",
			fmt.Sprintf("Podcast đã được thêm vào danh sách yêu thích"))

		c.JSON(http.StatusOK, gin.H{"message": "Đã yêu thích", "is_favorite": true})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn", "detail": err.Error()})
		return
	}

	// Bỏ yêu thích
	if err := config.DB.Delete(&fav).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bỏ yêu thích", "detail": err.Error()})
		return
	}

	config.DB.Model(&models.Podcast{}).Where("id = ?", podcastID).
		UpdateColumn("luot_yeu_thich", gorm.Expr("GREATEST(luot_yeu_thich - 1, 0)"))

	_ = services.CreateNotification(userID, podcastID, "favorite_removed",
		fmt.Sprintf("Podcast đã bị xóa khỏi danh sách yêu thích"))

	c.JSON(http.StatusOK, gin.H{"message": "Đã bỏ yêu thích", "is_favorite": false})
}

// ==================== LƯU PODCAST VÀO THƯ VIỆN ====================
func ToggleLuuPodcast(c *gin.Context) {
	podcastID := c.Param("id")
	if podcastID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu podcast_id"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn cần đăng nhập"})
		return
	}

	var save models.PodcastLuu
	err := config.DB.Where("nguoi_dung_id = ? AND podcast_id = ?", userID, podcastID).First(&save).Error

	if err == gorm.ErrRecordNotFound {
		// Lưu podcast
		save = models.PodcastLuu{
			ID:          uuid.New().String(),
			NguoiDungID: userID,
			PodcastID:   podcastID,
		}
		if err := config.DB.Create(&save).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu podcast", "detail": err.Error()})
			return
		}

		config.DB.Model(&models.Podcast{}).Where("id = ?", podcastID).
			UpdateColumn("luot_luu", gorm.Expr("luot_luu + 1"))

		_ = services.CreateNotification(userID, podcastID, "saved_added",
			fmt.Sprintf("Podcast đã được lưu vào thư viện"))

		c.JSON(http.StatusOK, gin.H{"message": "Đã lưu podcast", "is_saved": true})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn", "detail": err.Error()})
		return
	}

	// Bỏ lưu
	if err := config.DB.Delete(&save).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bỏ lưu", "detail": err.Error()})
		return
	}

	config.DB.Model(&models.Podcast{}).Where("id = ?", podcastID).
		UpdateColumn("luot_luu", gorm.Expr("GREATEST(luot_luu - 1, 0)"))

	_ = services.CreateNotification(userID, podcastID, "saved_removed",
		fmt.Sprintf("Podcast đã bị xóa khỏi thư viện"))

	c.JSON(http.StatusOK, gin.H{"message": "Đã bỏ lưu", "is_saved": false})
}

// ==================== LẤY DANH SÁCH YÊU THÍCH ====================
// ✅ FIXED: Thêm podcast_id vào response
func GetMyFavoritePodcasts(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn cần đăng nhập"})
		return
	}

	var list []models.PodcastYeuThich
	if err := config.DB.Preload("Podcast.TaiLieu").Preload("Podcast.DanhMuc").
		Where("nguoi_dung_id = ?", userID).
		Order("ngay_thich DESC").
		Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy danh sách yêu thích", "detail": err.Error()})
		return
	}

	var result []map[string]interface{}
	for _, item := range list {
		podcast := map[string]interface{}{
			// ✅ QUAN TRỌNG: Thêm podcast_id
			"podcast_id":        item.PodcastID, // ← Dòng này bị thiếu!
			"id":                item.Podcast.ID,
			"tieu_de":           item.Podcast.TieuDe,
			"mo_ta":             item.Podcast.MoTa,
			"hinh_anh_dai_dien": item.Podcast.HinhAnhDaiDien,
			"luot_yeu_thich":    item.Podcast.LuotYeuThich,
			"luot_luu":          item.Podcast.LuotLuu,
			"ngay_thich":        item.NgayThich,
		}

		if item.Podcast.TaiLieu.ID != "" {
			podcast["tom_tat"] = item.Podcast.TaiLieu.TomTat
		}

		if item.Podcast.DanhMuc.ID != "" {
			podcast["ten_danh_muc"] = item.Podcast.DanhMuc.TenDanhMuc
		}

		result = append(result, podcast)
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "total": len(result)})
}

// ==================== LẤY DANH SÁCH ĐÃ LƯU ====================
// ✅ FIXED: Thêm podcast_id vào response
func GetMySavedPodcasts(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn cần đăng nhập"})
		return
	}

	var list []models.PodcastLuu
	if err := config.DB.Preload("Podcast.TaiLieu").Preload("Podcast.DanhMuc").
		Where("nguoi_dung_id = ?", userID).
		Order("ngay_luu DESC").
		Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy danh sách đã lưu", "detail": err.Error()})
		return
	}

	var result []map[string]interface{}
	for _, item := range list {
		podcast := map[string]interface{}{
			// ✅ QUAN TRỌNG: Thêm podcast_id
			"podcast_id":        item.PodcastID, // ← Dòng này bị thiếu!
			"id":                item.Podcast.ID,
			"tieu_de":           item.Podcast.TieuDe,
			"mo_ta":             item.Podcast.MoTa,
			"hinh_anh_dai_dien": item.Podcast.HinhAnhDaiDien,
			"luot_yeu_thich":    item.Podcast.LuotYeuThich,
			"luot_luu":          item.Podcast.LuotLuu,
			"ngay_luu":          item.NgayLuu,
		}

		if item.Podcast.TaiLieu.ID != "" {
			podcast["tom_tat"] = item.Podcast.TaiLieu.TomTat
		}

		if item.Podcast.DanhMuc.ID != "" {
			podcast["ten_danh_muc"] = item.Podcast.DanhMuc.TenDanhMuc
		}

		result = append(result, podcast)
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "total": len(result)})
}

// ==================== LẤY LỊCH SỬ NGHE ====================
func GetMyListeningHistory(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn cần đăng nhập"})
		return
	}

	var history []models.LichSuNghe
	if err := config.DB.Preload("Podcast.TaiLieu").Preload("Podcast.DanhMuc").
		Where("nguoi_dung_id = ?", userID).
		Order("ngay_nghe DESC").
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy lịch sử nghe", "detail": err.Error()})
		return
	}

	type ListeningHistoryDTO struct {
		ID           string    `json:"id"`
		PodcastID    string    `json:"podcast_id"`
		TieuDe       string    `json:"tieu_de"`
		MoTa         string    `json:"mo_ta,omitempty"`
		HinhAnh      string    `json:"hinh_anh_dai_dien,omitempty"`
		TomTat       string    `json:"tom_tat,omitempty"`
		TenDanhMuc   string    `json:"ten_danh_muc,omitempty"`
		ViTriDaNghe  int       `json:"vi_tri_da_nghe"`
		ThoiGianNghe time.Time `json:"thoi_gian_nghe"`
		LuotYeuThich int       `json:"luot_yeu_thich"`
		LuotLuu      int       `json:"luot_luu"`
	}

	var result []ListeningHistoryDTO
	for _, h := range history {
		p := h.Podcast
		dto := ListeningHistoryDTO{
			ID:           h.ID,
			PodcastID:    p.ID,
			TieuDe:       p.TieuDe,
			MoTa:         p.MoTa,
			HinhAnh:      p.HinhAnhDaiDien,
			ViTriDaNghe:  h.ViTri,
			ThoiGianNghe: h.NgayNghe,
			LuotYeuThich: p.LuotYeuThich,
			LuotLuu:      p.LuotLuu,
		}

		if p.TaiLieu.ID != "" {
			dto.TomTat = p.TaiLieu.TomTat
		}

		if p.DanhMuc.ID != "" {
			dto.TenDanhMuc = p.DanhMuc.TenDanhMuc
		}

		result = append(result, dto)
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "total": len(result)})
}
