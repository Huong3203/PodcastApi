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

// ======================= Helper Functions =======================

func AttachSummary(db *gorm.DB, podcasts []models.Podcast) {
	for i := range podcasts {
		if podcasts[i].TailieuID != "" {
			var tl models.TaiLieu
			if err := db.First(&tl, "id = ?", podcasts[i].TailieuID).Error; err == nil {
				podcasts[i].TomTat = tl.TomTat
			}
		}
	}
}

func FormatSecondsToHHMMSS(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// ✅ Kiểm tra podcast có phải VIP không
func CheckPodcastVIPStatus(podcast *models.Podcast) bool {
	// Điều kiện 1: Podcast mới trong 7 ngày
	if time.Since(podcast.NgayTaoRa) <= 7*24*time.Hour {
		return true
	}

	// Điều kiện 2: Podcast có thời lượng > 3 phút (180 giây)
	if podcast.ThoiLuongGiay > 180 {
		return true
	}

	return false
}

// ✅ Kiểm tra user có VIP hợp lệ không
func IsUserVIP(user *models.NguoiDung) bool {
	if !user.VIP {
		return false
	}

	// Nếu không có ngày hết hạn = VIP vĩnh viễn
	if user.VIPExpires == nil {
		return true
	}

	// Kiểm tra VIP còn hạn
	return time.Now().Before(*user.VIPExpires)
}

// ======================= PUBLIC API =======================

// Xem danh sách podcast với VIP filter
func GetPodcast(c *gin.Context) {
	db := config.DB
	var podcasts []models.Podcast
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	search := c.Query("search")
	status := c.Query("status")
	categoryID := c.Query("category")
	sort := c.DefaultQuery("sort", "date")
	vipFilter := c.Query("vip") // "true", "false", hoặc ""

	query := db.Model(&models.Podcast{}).Preload("TaiLieu").Preload("DanhMuc")

	// Nếu không phải admin → chỉ lấy podcast có trạng thái "Bật"
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		query = query.Where("trang_thai = ?", "Bật")
	}

	if search != "" {
		query = query.Where("LOWER(tieu_de) LIKE ?", "%"+strings.ToLower(search)+"%")
	}

	if categoryID != "" {
		query = query.Where("danh_muc_id = ?", categoryID)
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
		orderBy = "luot_xem DESC"
	}

	query.Count(&total)
	query.Order(orderBy).Offset(offset).Limit(limit).Find(&podcasts)

	// ✅ Đánh dấu VIP và lọc nếu cần
	filteredPodcasts := []models.Podcast{}
	for i := range podcasts {
		podcasts[i].IsVIP = CheckPodcastVIPStatus(&podcasts[i])

		// Áp dụng filter VIP
		if vipFilter == "true" && podcasts[i].IsVIP {
			filteredPodcasts = append(filteredPodcasts, podcasts[i])
		} else if vipFilter == "false" && !podcasts[i].IsVIP {
			filteredPodcasts = append(filteredPodcasts, podcasts[i])
		} else if vipFilter == "" {
			filteredPodcasts = append(filteredPodcasts, podcasts[i])
		}
	}

	// Nếu có filter VIP, cập nhật lại total
	if vipFilter != "" {
		podcasts = filteredPodcasts
		total = int64(len(podcasts))
	}

	// Gán TomTat
	AttachSummary(db, podcasts)

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

// Tìm kiếm podcast với VIP marking
func SearchPodcast(c *gin.Context) {
	db := config.DB
	search := c.Query("q")
	status := c.Query("trang_thai")

	if search == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu từ khoá tìm kiếm"})
		return
	}

	var podcasts []models.Podcast
	query := db.Model(&models.Podcast{}).
		Where("LOWER(tieu_de) LIKE ? OR LOWER(mo_ta) LIKE ? OR LOWER(the_tag) LIKE ?",
			"%"+strings.ToLower(search)+"%",
			"%"+strings.ToLower(search)+"%",
			"%"+strings.ToLower(search)+"%",
		).
		Preload("TaiLieu").Preload("DanhMuc")

	if status != "" {
		query = query.Where("trang_thai = ?", status)
	}

	if err := query.Find(&podcasts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tìm kiếm podcast"})
		return
	}

	// ✅ Đánh dấu VIP
	for i := range podcasts {
		podcasts[i].IsVIP = CheckPodcastVIPStatus(&podcasts[i])
	}

	AttachSummary(db, podcasts)
	c.JSON(http.StatusOK, gin.H{"data": podcasts})
}

// ✅ Xem chi tiết podcast (WITH FULL VIP CHECK)
func GetPodcastByID(c *gin.Context) {
	db := config.DB
	id := c.Param("id")
	var podcast models.Podcast

	if err := db.Preload("TaiLieu").Preload("DanhMuc").First(&podcast, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy podcast"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy thông tin podcast"})
		}
		return
	}

	// ✅ Kiểm tra trạng thái VIP của podcast
	podcast.IsVIP = CheckPodcastVIPStatus(&podcast)

	// ✅ Kiểm tra quyền truy cập nếu podcast yêu cầu VIP
	if podcast.IsVIP {
		userIDStr := c.GetString("user_id")

		// Trường hợp 1: Chưa đăng nhập
		if userIDStr == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":           "VIP Required",
				"message":         "Podcast này yêu cầu VIP. Vui lòng đăng nhập và nâng cấp tài khoản VIP để nghe.",
				"is_vip_required": true,
				"requires_login":  true,
				"podcast_preview": gin.H{
					"id":                podcast.ID,
					"tieu_de":           podcast.TieuDe,
					"mo_ta":             podcast.MoTa,
					"hinh_anh_dai_dien": podcast.HinhAnhDaiDien,
					"thoi_luong_giay":   podcast.ThoiLuongGiay,
					"danh_muc":          podcast.DanhMuc,
					"is_vip":            true,
				},
			})
			return
		}

		// Trường hợp 2: Đã đăng nhập, kiểm tra VIP
		var user models.NguoiDung
		if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác thực người dùng"})
			return
		}

		// Trường hợp 3: Không có VIP hoặc VIP hết hạn
		if !IsUserVIP(&user) {
			vipExpired := user.VIP && user.VIPExpires != nil && time.Now().After(*user.VIPExpires)

			c.JSON(http.StatusForbidden, gin.H{
				"error":           "VIP Required",
				"message":         "Podcast này chỉ dành cho thành viên VIP. Vui lòng nâng cấp tài khoản để tiếp tục.",
				"is_vip_required": true,
				"vip_expired":     vipExpired,
				"podcast_preview": gin.H{
					"id":                podcast.ID,
					"tieu_de":           podcast.TieuDe,
					"mo_ta":             podcast.MoTa,
					"hinh_anh_dai_dien": podcast.HinhAnhDaiDien,
					"thoi_luong_giay":   podcast.ThoiLuongGiay,
					"danh_muc":          podcast.DanhMuc,
					"is_vip":            true,
				},
			})
			return
		}
	}

	// ✅ Tăng lượt xem (chỉ khi có quyền truy cập)
	db.Model(&podcast).UpdateColumn("luot_xem", gorm.Expr("luot_xem + ?", 1))

	// ✅ AUTO-SAVE LISTENING HISTORY (if user is logged in)
	userIDStr := c.GetString("user_id")
	if userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err == nil {
			podcastID, _ := uuid.Parse(id)

			var history models.ListeningHistory
			now := time.Now()

			result := db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).First(&history)

			if result.Error == gorm.ErrRecordNotFound {
				history = models.ListeningHistory{
					ID:        uuid.New(),
					UserID:    userID,
					PodcastID: podcastID,
				}
				db.Create(&history)
			} else if result.Error == nil {
				history.ListenedAt = now
				db.Save(&history)
			}
		}
	}

	// Gán TomTat
	if podcast.TailieuID != "" {
		podcast.TomTat = podcast.TaiLieu.TomTat
	}

	// Podcast liên quan
	var related []models.Podcast
	db.Preload("TaiLieu").Preload("DanhMuc").
		Where("danh_muc_id = ? AND id != ?", podcast.DanhMucID, podcast.ID).
		Order("ngay_tao_ra DESC").Limit(5).Find(&related)

	// ✅ Đánh dấu VIP cho podcast liên quan
	for i := range related {
		related[i].IsVIP = CheckPodcastVIPStatus(&related[i])
	}
	AttachSummary(db, related)

	c.JSON(http.StatusOK, gin.H{
		"data":    podcast,
		"suggest": related,
	})
}

// Lấy danh sách podcast đang tắt
func GetDisabledPodcasts(c *gin.Context) {
	db := config.DB
	var podcasts []models.Podcast

	if err := db.Where("trang_thai = ?", "Tắt").
		Preload("TaiLieu").Preload("DanhMuc").
		Order("ngay_tao_ra DESC").Find(&podcasts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Lỗi khi lấy danh sách podcast bị tắt",
			"detail": err.Error(),
		})
		return
	}

	// ✅ Đánh dấu VIP
	for i := range podcasts {
		podcasts[i].IsVIP = CheckPodcastVIPStatus(&podcasts[i])
	}

	AttachSummary(db, podcasts)

	c.JSON(http.StatusOK, gin.H{
		"count": len(podcasts),
		"data":  podcasts,
	})
}

// Tạo podcast với upload tài liệu + audio
func CreatePodcastWithUpload(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn phải đăng nhập"})
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

	// ✅ Đánh dấu VIP cho podcast mới tạo
	podcast.IsVIP = CheckPodcastVIPStatus(&podcast)

	message := fmt.Sprintf("Người dùng %s đã tạo podcast: %s", userID, tieuDe)
	services.CreateNotification(userID, podcast.ID, "create_podcast", message)

	c.JSON(http.StatusOK, gin.H{
		"message": "Tạo podcast thành công",
		"podcast": podcast,
	})
}

// Cập nhật podcast
func UpdatePodcast(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
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
		changes = append(changes, "mô tả")
		podcast.MoTa = moTa
	}
	if theTag != "" && theTag != podcast.TheTag {
		changes = append(changes, "tag")
		podcast.TheTag = theTag
	}
	if danhMucID != "" && danhMucID != podcast.DanhMucID {
		changes = append(changes, "danh mục")
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

	if len(changes) > 0 {
		message := fmt.Sprintf("Podcast %s đã được cập nhật: %v", podcast.TieuDe, changes)
		services.CreateNotification("", podcast.ID, "update_podcast", message)
	}

	db.Preload("TaiLieu").Preload("DanhMuc").First(&podcast, "id = ?", podcastID)

	// ✅ Đánh dấu VIP
	podcast.IsVIP = CheckPodcastVIPStatus(&podcast)
	AttachSummary(db, []models.Podcast{podcast})

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật podcast thành công",
		"podcast": podcast,
	})
}

// Gợi ý podcast tương tự
func GetRecommendedPodcasts(c *gin.Context) {
	db := config.DB
	podcastID := c.Param("id")

	var current models.Podcast
	if err := db.First(&current, "id = ?", podcastID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy podcast"})
		return
	}

	type PodcastWithStats struct {
		models.Podcast
		AvgRating  float64 `json:"avg_rating"`
		TotalVotes int64   `json:"total_votes"`
		TomTat     string  `json:"tom_tat"`
	}

	var recommendations []PodcastWithStats

	if err := db.Table("podcasts p").
		Select(`p.*, COALESCE(AVG(d.sao),0) AS avg_rating, COUNT(d.id) AS total_votes`).
		Joins("LEFT JOIN danh_gias d ON d.podcast_id = p.id").
		Where("p.danh_muc_id = ? AND p.id != ? AND p.trang_thai = ?", current.DanhMucID, current.ID, "Bật").
		Group("p.id").
		Order("avg_rating DESC, p.luot_xem DESC, p.ngay_tao_ra DESC").
		Limit(6).
		Scan(&recommendations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách đề xuất"})
		return
	}

	// ✅ Đánh dấu VIP và gán TomTat
	for i := range recommendations {
		var tl models.TaiLieu
		if err := db.First(&tl, "id = ?", recommendations[i].TailieuID).Error; err == nil {
			recommendations[i].TomTat = tl.TomTat
		}
		recommendations[i].IsVIP = CheckPodcastVIPStatus(&recommendations[i].Podcast)
	}

	// fallback nếu không có cùng danh mục
	if len(recommendations) == 0 {
		db.Table("podcasts p").
			Select(`p.*, COALESCE(AVG(d.sao),0) AS avg_rating, COUNT(d.id) AS total_votes`).
			Joins("LEFT JOIN danh_gias d ON d.podcast_id = p.id").
			Where("p.id != ? AND p.trang_thai = ?", current.ID, "Bật").
			Group("p.id").
			Order("avg_rating DESC, total_votes DESC").
			Limit(6).
			Scan(&recommendations)

		for i := range recommendations {
			var tl models.TaiLieu
			if err := db.First(&tl, "id = ?", recommendations[i].TailieuID).Error; err == nil {
				recommendations[i].TomTat = tl.TomTat
			}
			recommendations[i].IsVIP = CheckPodcastVIPStatus(&recommendations[i].Podcast)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": recommendations})
}
