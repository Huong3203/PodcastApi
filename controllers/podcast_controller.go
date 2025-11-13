package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/services"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Xem danh sách podcast
func GetPodcast(c *gin.Context) {
	var podcasts []models.Podcast
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	search := c.Query("search")
	status := c.Query("status")
	categoryID := c.Query("category")
	sort := c.DefaultQuery("sort", "date")

	query := config.DB.Model(&models.Podcast{})

	// Nếu không phải admin → chỉ lấy podcast có trạng thái "Bật"
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		query = query.Where("trang_thai = ?", "Bật")
	}

	if search != "" {
		query = query.Where("LOWER(tieu_de) LIKE ?", "%"+strings.ToLower(search)+"%")
	}

	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	if status != "" && role == "admin" {
		switch status {
		case "Bật":
			query = query.Where("trang_thai = ?", "Bật")
		case "Tắt":
			query = query.Where("trang_thai = ?", "Tắt")
		}
	}

	// Sắp xếp
	orderBy := "ngay_tao_ra DESC"
	if sort == "views" {
		orderBy = "views DESC"
	}

	query.Count(&total)
	query.Order(orderBy).Offset(offset).Limit(limit).Find(&podcasts)

	c.JSON(http.StatusOK, gin.H{
		"data": podcasts,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// Tìm kiếm podcast
func SearchPodcast(c *gin.Context) {
	search := c.Query("q")
	status := c.Query("trang_thai")

	if search == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu từ khoá tìm kiếm"})
		return
	}

	var podcasts []models.Podcast
	query := config.DB.Model(&models.Podcast{}).
		Where("LOWER(tieu_de) LIKE ? OR LOWER(mo_ta) LIKE ? OR LOWER(the_tag) LIKE ?",
			"%"+strings.ToLower(search)+"%",
			"%"+strings.ToLower(search)+"%",
			"%"+strings.ToLower(search)+"%",
		)

	if status != "" {
		query = query.Where("trang_thai = ?", status)
	}

	query = query.Preload("TaiLieu").Preload("DanhMuc")

	if err := query.Find(&podcasts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tìm kiếm podcast"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": podcasts})
}

// Xem chi tiết podcast
func GetPodcastByID(c *gin.Context) {
	id := c.Param("id")
	var podcast models.Podcast

	if err := config.DB.First(&podcast, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy podcast"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy thông tin podcast"})
		}
		return
	}

	// Tăng lượt xem
	config.DB.Model(&podcast).UpdateColumn("luot_xem", gorm.Expr("luot_xem + ?", 1))

	// Podcast liên quan
	var related []models.Podcast
	config.DB.Where("danh_muc_id = ? AND id != ?", podcast.DanhMucID, podcast.ID).
		Order("ngay_tao_ra DESC").Limit(5).Find(&related)

	c.JSON(http.StatusOK, gin.H{
		"data":    podcast,
		"suggest": related,
	})
}

// Lấy danh sách podcast đang tắt
func GetDisabledPodcasts(c *gin.Context) {
	var podcasts []models.Podcast

	if err := config.DB.
		Where("trang_thai = ?", "Tắt").
		Order("ngay_tao_ra DESC").
		Find(&podcasts).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Lỗi khi lấy danh sách podcast bị tắt",
			"detail": err.Error(), // hiện chi tiết lỗi để debug
		})
		return
	}

	if len(podcasts) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "Không có podcast nào đang bị tắt",
			"data":    []models.Podcast{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(podcasts),
		"data":  podcasts,
	})
}

// Tạo podcast (yêu cầu đăng nhập)
func CreatePodcastWithUpload(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn phải đăng nhập để thực hiện hành động này"})
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	userID := c.GetString("user_id")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có file đính kèm"})
		return
	}

	tieuDe := c.PostForm("tieu_de")
	danhMucID := c.PostForm("danh_muc_id")
	if tieuDe == "" || danhMucID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu tiêu đề hoặc danh mục"})
		return
	}

	moTa := c.PostForm("mo_ta")
	hinhAnh := ""
	if hinhAnhFile, err := c.FormFile("hinh_anh_dai_dien"); err == nil {
		imageURL, err := utils.UploadImageToSupabase(hinhAnhFile, uuid.New().String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể upload hình ảnh", "details": err.Error()})
			return
		}
		hinhAnh = imageURL
	}

	theTag := c.PostForm("the_tag")
	voice := c.DefaultPostForm("voice", "vi-VN-Chirp3-HD-Puck")
	speakingRateStr := c.DefaultPostForm("speaking_rate", "1.0")
	rateValue, _ := strconv.ParseFloat(speakingRateStr, 64)
	if rateValue <= 0 {
		rateValue = 1.0
	}

	authHeader := c.GetHeader("Authorization")
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header không hợp lệ"})
		return
	}
	token := parts[1]

	respData, err := services.CallUploadDocumentAPI(file, userID, token, voice, rateValue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi gọi UploadDocument", "details": err.Error()})
		return
	}

	taiLieuRaw, ok := respData["tai_lieu"]
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy dữ liệu tài liệu từ UploadDocument"})
		return
	}

	taiLieuMap, ok := taiLieuRaw.(map[string]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu tài liệu không đúng định dạng"})
		return
	}

	audioURL, _ := respData["audio_url"].(string)
	taiLieuID, _ := taiLieuMap["id"].(string)

	durationFloat, _ := services.GetMP3DurationFromURL(audioURL)
	totalSeconds := int(durationFloat)

	podcast := models.Podcast{
		ID:             uuid.New().String(),
		TailieuID:      taiLieuID,
		TieuDe:         tieuDe,
		MoTa:           moTa,
		DuongDanAudio:  audioURL,
		ThoiLuongGiay:  totalSeconds,
		HinhAnhDaiDien: hinhAnh,
		DanhMucID:      danhMucID,
		TrangThai:      "Tắt",
		NguoiTao:       userID,
		TheTag:         theTag,
		LuotXem:        0,
	}

	if err := db.Create(&podcast).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo podcast", "details": err.Error()})
		return
	}

	// 🔹 Tạo thông báo realtime
	message := fmt.Sprintf("Người dùng %s đã tạo podcast: %s", userID, tieuDe)
	if err := services.CreateNotification(userID, podcast.ID, "create_podcast", message); err != nil {
		fmt.Println("Lỗi khi tạo thông báo:", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tạo podcast thành công",
		"podcast": podcast,
	})
}

// Cập nhật podcast (Admin)
func UpdatePodcast(c *gin.Context) {
	if role, _ := c.Get("vai_tro"); role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền chỉnh sửa podcast"})
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	podcastID := c.Param("id")

	var podcast models.Podcast
	if err := db.First(&podcast, "id = ?", podcastID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Podcast không tồn tại"})
		return
	}

	tieuDe := c.PostForm("tieu_de")
	moTa := c.PostForm("mo_ta")
	theTag := c.PostForm("the_tag")
	danhMucID := c.PostForm("danh_muc_id")
	trangThai := c.PostForm("trang_thai")

	changes := []string{}

	if tieuDe != "" && tieuDe != podcast.TieuDe {
		changes = append(changes, fmt.Sprintf("tiêu đề: %s → %s", podcast.TieuDe, tieuDe))
		podcast.TieuDe = tieuDe
	}
	if moTa != "" && moTa != podcast.MoTa {
		changes = append(changes, fmt.Sprintf("mô tả"))
		podcast.MoTa = moTa
	}
	if theTag != "" && theTag != podcast.TheTag {
		changes = append(changes, fmt.Sprintf("tag"))
		podcast.TheTag = theTag
	}
	if danhMucID != "" && danhMucID != podcast.DanhMucID {
		changes = append(changes, fmt.Sprintf("danh mục"))
		podcast.DanhMucID = danhMucID
	}
	if trangThai != "" && trangThai != podcast.TrangThai {
		changes = append(changes, fmt.Sprintf("trạng thái: %s → %s", podcast.TrangThai, trangThai))
		podcast.TrangThai = trangThai
		if trangThai == "Bật" {
			now := time.Now()
			podcast.NgayXuatBan = &now
		}
	}

	if hinhAnhFile, err := c.FormFile("hinh_anh_dai_dien"); err == nil {
		if imageURL, err := utils.UploadImageToSupabase(hinhAnhFile, uuid.New().String()); err == nil {
			podcast.HinhAnhDaiDien = imageURL
			changes = append(changes, "hình ảnh đại diện")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể upload hình ảnh"})
			return
		}
	}

	if err := db.Save(&podcast).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật podcast"})
		return
	}

	// 🔹 Tạo thông báo realtime
	if len(changes) > 0 {
		message := fmt.Sprintf("Podcast %s đã được cập nhật: %v", podcast.TieuDe, changes)
		if err := services.CreateNotification("", podcast.ID, "update_podcast", message); err != nil {
			fmt.Println("Lỗi khi tạo thông báo:", err)
		}
	}

	db.Preload("TaiLieu.NguoiDung").Preload("DanhMuc").First(&podcast, "id = ?", podcastID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật podcast thành công",
		"podcast": podcast,
	})
}

//  Gợi ý podcast tương tự (recommendations)

func GetRecommendedPodcasts(c *gin.Context) {
	db := config.DB
	podcastID := c.Param("id")

	// Lấy podcast gốc để tìm danh mục
	var current models.Podcast
	if err := db.First(&current, "id = ?", podcastID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy podcast"})
		return
	}

	type PodcastWithStats struct {
		models.Podcast
		AvgRating  float64 `json:"avg_rating"`
		TotalVotes int64   `json:"total_votes"`
	}

	var recommendations []PodcastWithStats

	// Lấy các podcast cùng danh mục, khác ID hiện tại
	if err := db.Table("podcasts p").
		Select(`
			p.*, 
			COALESCE(AVG(d.sao), 0) AS avg_rating, 
			COUNT(d.id) AS total_votes
		`).
		Joins("LEFT JOIN danh_gias d ON d.podcast_id = p.id").
		Where("p.danh_muc_id = ? AND p.id != ? AND p.trang_thai = ?", current.DanhMucID, current.ID, "Bật").
		Group("p.id").
		Order("avg_rating DESC, p.luot_xem DESC, p.ngay_tao_ra DESC").
		Limit(6).
		Scan(&recommendations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách đề xuất"})
		return
	}

	// Nếu không có cùng danh mục → fallback: lấy ngẫu nhiên 6 podcast nổi bật
	if len(recommendations) == 0 {
		db.Table("podcasts p").
			Select(`
				p.*, 
				COALESCE(AVG(d.sao), 0) AS avg_rating, 
				COUNT(d.id) AS total_votes
			`).
			Joins("LEFT JOIN danh_gias d ON d.podcast_id = p.id").
			Where("p.id != ? AND p.trang_thai = ?", current.ID, "Bật").
			Group("p.id").
			Order("avg_rating DESC, total_votes DESC").
			Limit(6).
			Scan(&recommendations)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": recommendations,
	})
}

// Format thời lượng

func FormatSecondsToHHMMSS(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
