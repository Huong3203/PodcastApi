package controllers

import (
	"fmt"
	"log"
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
func UpdatePodcastVIPStatus(db *gorm.DB, podcast *models.Podcast) {
	if db == nil {
		log.Println("Warning: DB is nil in UpdatePodcastVIPStatus")
		return
	}

	shouldBeVIP := CheckPodcastVIPStatus(podcast)
	if podcast.IsVIP != shouldBeVIP {
		if err := db.Model(podcast).Update("is_vip", shouldBeVIP).Error; err != nil {
			log.Printf("Warning: Failed to update VIP status: %v", err)
			return
		}
		podcast.IsVIP = shouldBeVIP
	}
}

func BatchUpdateVIPStatus(db *gorm.DB, podcasts []models.Podcast) {
	if db == nil {
		log.Println("Warning: DB is nil in BatchUpdateVIPStatus")
		return
	}
	for i := range podcasts {
		UpdatePodcastVIPStatus(db, &podcasts[i])
	}
}

func AttachSummary(db *gorm.DB, podcasts []models.Podcast) {
	if db == nil {
		return
	}
	for i := range podcasts {
		if podcasts[i].TailieuID != "" {
			var tl models.TaiLieu
			if err := db.First(&tl, "id = ?", podcasts[i].TailieuID).Error; err == nil {
				podcasts[i].TomTat = tl.TomTat
			}
		}
	}
}

func CheckPodcastVIPStatus(podcast *models.Podcast) bool {
	if time.Since(podcast.NgayTaoRa) <= 7*24*time.Hour {
		return true
	}
	if podcast.ThoiLuongGiay > 180 {
		return true
	}
	return false
}

func IsUserVIP(user *models.NguoiDung) bool {
	if !user.VIP {
		return false
	}
	if user.VIPExpires == nil {
		return true
	}
	return time.Now().Before(*user.VIPExpires)
}

// ======================= PUBLIC API =======================

// ✅ FIX: GetPodcast - Lấy DB từ context thay vì config.DB
func GetPodcast(c *gin.Context) {
	// ✅ FIX: Lấy DB từ middleware context
	dbInterface, exists := c.Get("db")
	if !exists {
		log.Println("ERROR: Database not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Lỗi hệ thống: không thể kết nối database",
		})
		return
	}

	db, ok := dbInterface.(*gorm.DB)
	if !ok {
		log.Println("ERROR: Database type assertion failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Lỗi hệ thống: database không hợp lệ",
		})
		return
	}

	// Kiểm tra DB connection
	if db == nil {
		log.Println("ERROR: DB is nil after assertion")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể kết nối database",
		})
		return
	}

	var podcasts []models.Podcast
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	search := c.Query("search")
	status := c.Query("status")
	categoryID := c.Query("category")
	sort := c.DefaultQuery("sort", "date")
	vipFilter := c.Query("vip")

	query := db.Model(&models.Podcast{}).Preload("TaiLieu").Preload("DanhMuc")

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
		query = query.Where("trang_thai = ?", status)
	}

	if vipFilter == "true" {
		query = query.Where("is_vip = ?", true)
	} else if vipFilter == "false" {
		query = query.Where("is_vip = ?", false)
	}

	orderBy := "ngay_tao_ra DESC"
	if sort == "views" {
		orderBy = "luot_xem DESC"
	}

	if err := query.Count(&total).Error; err != nil {
		log.Printf("ERROR: Failed to count podcasts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Lỗi khi đếm podcast",
			"details": err.Error(),
		})
		return
	}

	if err := query.Order(orderBy).Offset(offset).Limit(limit).Find(&podcasts).Error; err != nil {
		log.Printf("ERROR: Failed to fetch podcasts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Lỗi khi lấy danh sách podcast",
			"details": err.Error(),
		})
		return
	}

	// Background VIP sync
	go BatchUpdateVIPStatus(db, podcasts)
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

// ✅ FIX: CreatePodcastWithUpload - Improved error handling
func CreatePodcastWithUpload(c *gin.Context) {
	role, exists := c.Get("vai_tro")
	if !exists {
		log.Println("ERROR: Role not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn phải đăng nhập"})
		return
	}

	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền tạo podcast"})
		return
	}

	dbInterface, exists := c.Get("db")
	if !exists {
		log.Println("ERROR: Database not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống: không thể kết nối database"})
		return
	}

	db, ok := dbInterface.(*gorm.DB)
	if !ok || db == nil {
		log.Println("ERROR: Database type assertion failed or DB is nil")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống: database không hợp lệ"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		log.Println("ERROR: User ID not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không thể xác định user"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("ERROR: File upload failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có file đính kèm"})
		return
	}

	tieuDe := strings.TrimSpace(c.PostForm("tieu_de"))
	danhMucID := strings.TrimSpace(c.PostForm("danh_muc_id"))

	if tieuDe == "" || danhMucID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu tiêu đề hoặc danh mục"})
		return
	}

	// Validate category exists
	var category models.DanhMuc
	if err := db.First(&category, "id = ?", danhMucID).Error; err != nil {
		log.Printf("ERROR: Category not found: %s - %v", danhMucID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Danh mục không tồn tại"})
		return
	}

	moTa := c.PostForm("mo_ta")
	hinhAnh := ""

	if hinhAnhFile, err := c.FormFile("hinh_anh_dai_dien"); err == nil {
		imageURL, err := utils.UploadImageToSupabase(hinhAnhFile, uuid.New().String())
		if err != nil {
			log.Printf("ERROR: Image upload failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Không thể upload hình ảnh",
				"details": err.Error(),
			})
			return
		}
		hinhAnh = imageURL
	}

	theTag := c.PostForm("the_tag")
	voice := c.DefaultPostForm("voice", "vi-VN-Chirp3-HD-Puck")
	speakingRateStr := c.DefaultPostForm("speaking_rate", "1.0")

	rateValue, err := strconv.ParseFloat(speakingRateStr, 64)
	if err != nil || rateValue <= 0 {
		rateValue = 1.0
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header không hợp lệ"})
		return
	}
	token := parts[1]

	log.Printf("Calling UploadDocument API for user: %s", userID)
	respData, err := services.CallUploadDocumentAPI(file, userID, token, voice, rateValue)
	if err != nil {
		log.Printf("ERROR: UploadDocument API failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Lỗi khi xử lý tài liệu",
			"details": err.Error(),
		})
		return
	}

	taiLieuRaw, ok := respData["tai_lieu"]
	if !ok {
		log.Println("ERROR: tai_lieu not found in response")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy dữ liệu tài liệu từ UploadDocument",
		})
		return
	}

	taiLieuMap, ok := taiLieuRaw.(map[string]interface{})
	if !ok {
		log.Println("ERROR: tai_lieu format invalid")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Dữ liệu tài liệu không đúng định dạng",
		})
		return
	}

	audioURL, _ := respData["audio_url"].(string)
	if audioURL == "" {
		log.Println("ERROR: audio_url not found")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy đường dẫn audio",
		})
		return
	}

	taiLieuID, _ := taiLieuMap["id"].(string)
	if taiLieuID == "" {
		log.Println("ERROR: tai_lieu ID not found")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy ID tài liệu",
		})
		return
	}

	durationFloat, err := services.GetMP3DurationFromURL(audioURL)
	if err != nil {
		log.Printf("WARNING: Failed to get audio duration: %v", err)
		durationFloat = 0
	}
	totalSeconds := int(durationFloat)

	isVIP := true
	if totalSeconds > 180 {
		isVIP = true
	}

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
		IsVIP:          isVIP,
		NgayTaoRa:      time.Now(),
	}

	if err := db.Create(&podcast).Error; err != nil {
		log.Printf("ERROR: Failed to create podcast: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Không thể tạo podcast",
			"details": err.Error(),
		})
		return
	}

	go func() {
		message := fmt.Sprintf("Admin %s đã tạo podcast: %s", userID, tieuDe)
		if err := services.CreateNotification(userID, podcast.ID, "create_podcast", message); err != nil {
			log.Printf("WARNING: Failed to create notification: %v", err)
		}
	}()

	log.Printf("SUCCESS: Created podcast %s (ID: %s)", tieuDe, podcast.ID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Tạo podcast thành công",
		"podcast": podcast,
	})
}

// ✅ FIX: GetPodcastByID - Use DB from context
func GetPodcastByID(c *gin.Context) {
	dbInterface, exists := c.Get("db")
	var db *gorm.DB

	if exists {
		db, _ = dbInterface.(*gorm.DB)
	}

	// Fallback to config.DB if not in context
	if db == nil {
		db = config.DB
	}

	if db == nil {
		log.Println("ERROR: No database connection available")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kết nối database"})
		return
	}

	id := c.Param("id")
	var podcast models.Podcast

	if err := db.Preload("TaiLieu").Preload("DanhMuc").First(&podcast, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy podcast"})
		} else {
			log.Printf("ERROR: Failed to fetch podcast %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy thông tin podcast"})
		}
		return
	}

	UpdatePodcastVIPStatus(db, &podcast)

	role, _ := c.Get("vai_tro")

	if role != "admin" && podcast.IsVIP {
		userIDStr := c.GetString("user_id")

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

		var user models.NguoiDung
		if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
			log.Printf("ERROR: Failed to fetch user %s: %v", userIDStr, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác thực người dùng"})
			return
		}

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

	db.Model(&podcast).UpdateColumn("luot_xem", gorm.Expr("luot_xem + ?", 1))

	userIDStr := c.GetString("user_id")
	if userIDStr != "" {
		var history models.LichSuNghe
		now := time.Now()

		result := db.Where("nguoi_dung_id = ? AND podcast_id = ?", userIDStr, id).First(&history)

		if result.Error == gorm.ErrRecordNotFound {
			history = models.LichSuNghe{
				ID:          uuid.New().String(),
				NguoiDungID: userIDStr,
				PodcastID:   id,
				ViTri:       0,
				NgayNghe:    now,
			}
			if err := db.Create(&history).Error; err != nil {
				log.Printf("WARNING: Failed to create listening history: %v", err)
			}
		} else if result.Error == nil {
			if err := db.Model(&history).Update("ngay_nghe", now).Error; err != nil {
				log.Printf("WARNING: Failed to update listening history: %v", err)
			}
		}
	}

	if podcast.TailieuID != "" {
		podcast.TomTat = podcast.TaiLieu.TomTat
	}

	var related []models.Podcast
	db.Preload("TaiLieu").Preload("DanhMuc").
		Where("danh_muc_id = ? AND id != ?", podcast.DanhMucID, podcast.ID).
		Order("ngay_tao_ra DESC").Limit(5).Find(&related)

	go BatchUpdateVIPStatus(db, related)
	AttachSummary(db, related)

	c.JSON(http.StatusOK, gin.H{
		"data":    podcast,
		"suggest": related,
	})
}

// ✅ Các hàm còn lại với fallback DB handling
func SearchPodcast(c *gin.Context) {
	dbInterface, _ := c.Get("db")
	db, _ := dbInterface.(*gorm.DB)
	if db == nil {
		db = config.DB
	}

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

	for i := range podcasts {
		podcasts[i].IsVIP = CheckPodcastVIPStatus(&podcasts[i])
	}

	AttachSummary(db, podcasts)
	c.JSON(http.StatusOK, gin.H{"data": podcasts})
}

func GetDisabledPodcasts(c *gin.Context) {
	dbInterface, _ := c.Get("db")
	db, _ := dbInterface.(*gorm.DB)
	if db == nil {
		db = config.DB
	}

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

	for i := range podcasts {
		podcasts[i].IsVIP = CheckPodcastVIPStatus(&podcasts[i])
	}

	AttachSummary(db, podcasts)

	c.JSON(http.StatusOK, gin.H{
		"count": len(podcasts),
		"data":  podcasts,
	})
}

func GetRecommendedPodcasts(c *gin.Context) {
	dbInterface, _ := c.Get("db")
	db, _ := dbInterface.(*gorm.DB)
	if db == nil {
		db = config.DB
	}

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

	for i := range recommendations {
		var tl models.TaiLieu
		if err := db.First(&tl, "id = ?", recommendations[i].TailieuID).Error; err == nil {
			recommendations[i].TomTat = tl.TomTat
		}
		recommendations[i].IsVIP = CheckPodcastVIPStatus(&recommendations[i].Podcast)
	}

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

func GetFeaturedPodcasts(c *gin.Context) {
	dbInterface, _ := c.Get("db")
	db, _ := dbInterface.(*gorm.DB)
	if db == nil {
		db = config.DB
	}

	var podcasts []models.Podcast
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	if err := db.Where("trang_thai = ? AND ngay_tao_ra >= ?", "Bật", thirtyDaysAgo).
		Preload("TaiLieu").
		Preload("DanhMuc").
		Order("luot_xem DESC").
		Limit(10).
		Find(&podcasts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy danh sách podcast nổi bật",
		})
		return
	}

	for i := range podcasts {
		podcasts[i].IsVIP = CheckPodcastVIPStatus(&podcasts[i])
	}

	AttachSummary(db, podcasts)

	c.JSON(http.StatusOK, gin.H{
		"data": podcasts,
	})
}

func UpdatePodcast(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền chỉnh sửa podcast"})
		return
	}

	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống"})
		return
	}

	db, ok := dbInterface.(*gorm.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống"})
		return
	}

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
	vipStatus := c.PostForm("is_vip")

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

	if vipStatus != "" {
		newVIP := vipStatus == "true"
		if newVIP != podcast.IsVIP {
			changes = append(changes, fmt.Sprintf("VIP: %v → %v", podcast.IsVIP, newVIP))
			podcast.IsVIP = newVIP
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
		log.Printf("ERROR: Failed to update podcast: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật podcast"})
		return
	}

	if len(changes) > 0 {
		go func() {
			message := fmt.Sprintf("Podcast %s đã được cập nhật: %v", podcast.TieuDe, changes)
			services.CreateNotification("", podcast.ID, "update_podcast", message)
		}()
	}

	db.Preload("TaiLieu").Preload("DanhMuc").First(&podcast, "id = ?", podcastID)
	AttachSummary(db, []models.Podcast{podcast})

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật podcast thành công",
		"podcast": podcast,
	})
}

func TogglePodcastVIPStatus(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền thay đổi trạng thái VIP"})
		return
	}

	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống"})
		return
	}

	db, ok := dbInterface.(*gorm.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống"})
		return
	}

	podcastID := c.Param("id")

	var podcast models.Podcast
	if err := db.First(&podcast, "id = ?", podcastID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Podcast không tồn tại"})
		return
	}

	newVIPStatus := !podcast.IsVIP
	if err := db.Model(&podcast).Update("is_vip", newVIPStatus).Error; err != nil {
		log.Printf("ERROR: Failed to toggle VIP status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái VIP"})
		return
	}

	podcast.IsVIP = newVIPStatus

	go func() {
		message := fmt.Sprintf("Admin đã thay đổi trạng thái VIP của podcast '%s' thành: %v", podcast.TieuDe, newVIPStatus)
		services.CreateNotification("", podcast.ID, "update_vip", message)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Đã %s trạng thái VIP", map[bool]string{true: "bật", false: "tắt"}[newVIPStatus]),
		"podcast": podcast,
	})
}

func SyncAllVIPStatus(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền sync VIP"})
		return
	}

	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống"})
		return
	}

	db, ok := dbInterface.(*gorm.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống"})
		return
	}

	var podcasts []models.Podcast
	if err := db.Find(&podcasts).Error; err != nil {
		log.Printf("ERROR: Failed to fetch podcasts for sync: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách podcast"})
		return
	}

	updated := 0
	for i := range podcasts {
		oldStatus := podcasts[i].IsVIP
		shouldBeVIP := CheckPodcastVIPStatus(&podcasts[i])

		if oldStatus != shouldBeVIP {
			if err := db.Model(&podcasts[i]).Update("is_vip", shouldBeVIP).Error; err != nil {
				log.Printf("WARNING: Failed to update VIP for podcast %s: %v", podcasts[i].ID, err)
				continue
			}
			updated++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Đã đồng bộ trạng thái VIP",
		"total_podcasts": len(podcasts),
		"updated":        updated,
	})
}

// package controllers

// import (
// 	"fmt"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/Huong3203/APIPodcast/config"
// 	"github.com/Huong3203/APIPodcast/models"
// 	"github.com/Huong3203/APIPodcast/services"
// 	"github.com/Huong3203/APIPodcast/utils"
// 	"github.com/gin-gonic/gin"
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// // ======================= Helper Functions =======================
// // ✅ Helper: Tính toán và UPDATE trạng thái VIP vào database
// func UpdatePodcastVIPStatus(db *gorm.DB, podcast *models.Podcast) {
// 	shouldBeVIP := CheckPodcastVIPStatus(podcast)

// 	// Chỉ update nếu trạng thái thay đổi
// 	if podcast.IsVIP != shouldBeVIP {
// 		db.Model(podcast).Update("is_vip", shouldBeVIP)
// 		podcast.IsVIP = shouldBeVIP
// 	}
// }

// // ✅ Helper: Batch update VIP status cho nhiều podcasts
// func BatchUpdateVIPStatus(db *gorm.DB, podcasts []models.Podcast) {
// 	for i := range podcasts {
// 		UpdatePodcastVIPStatus(db, &podcasts[i])
// 	}
// }

// func AttachSummary(db *gorm.DB, podcasts []models.Podcast) {
// 	for i := range podcasts {
// 		if podcasts[i].TailieuID != "" {
// 			var tl models.TaiLieu
// 			if err := db.First(&tl, "id = ?", podcasts[i].TailieuID).Error; err == nil {
// 				podcasts[i].TomTat = tl.TomTat
// 			}
// 		}
// 	}
// }

// func FormatSecondsToHHMMSS(seconds int) string {
// 	h := seconds / 3600
// 	m := (seconds % 3600) / 60
// 	s := seconds % 60
// 	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
// }

// // ✅ Kiểm tra podcast có phải VIP không
// func CheckPodcastVIPStatus(podcast *models.Podcast) bool {
// 	// Điều kiện 1: Podcast mới trong 7 ngày
// 	if time.Since(podcast.NgayTaoRa) <= 7*24*time.Hour {
// 		return true
// 	}

// 	// Điều kiện 2: Podcast có thời lượng > 3 phút (180 giây)
// 	if podcast.ThoiLuongGiay > 180 {
// 		return true
// 	}

// 	return false
// }

// // ✅ Kiểm tra user có VIP hợp lệ không
// func IsUserVIP(user *models.NguoiDung) bool {
// 	if !user.VIP {
// 		return false
// 	}

// 	// Nếu không có ngày hết hạn = VIP vĩnh viễn
// 	if user.VIPExpires == nil {
// 		return true
// 	}

// 	// Kiểm tra VIP còn hạn
// 	return time.Now().Before(*user.VIPExpires)
// }

// // ======================= PUBLIC API =======================

// // ✅ Tạo podcast - CHỈ ADMIN
// func CreatePodcastWithUpload(c *gin.Context) {
// 	// ✅ FIX: Kiểm tra role đúng cách
// 	role, exists := c.Get("vai_tro")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn phải đăng nhập"})
// 		return
// 	}

// 	// ✅ Chỉ admin mới được tạo podcast
// 	if role != "admin" {
// 		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền tạo podcast"})
// 		return
// 	}

// 	// ✅ FIX: Lấy db từ middleware
// 	db := c.MustGet("db").(*gorm.DB)
// 	userID := c.GetString("user_id")

// 	if userID == "" {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không thể xác định user"})
// 		return
// 	}

// 	file, err := c.FormFile("file")
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có file đính kèm"})
// 		return
// 	}

// 	tieuDe := c.PostForm("tieu_de")
// 	danhMucID := c.PostForm("danh_muc_id")
// 	if tieuDe == "" || danhMucID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu tiêu đề hoặc danh mục"})
// 		return
// 	}

// 	moTa := c.PostForm("mo_ta")
// 	hinhAnh := ""
// 	if hinhAnhFile, err := c.FormFile("hinh_anh_dai_dien"); err == nil {
// 		imageURL, err := utils.UploadImageToSupabase(hinhAnhFile, uuid.New().String())
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể upload hình ảnh", "details": err.Error()})
// 			return
// 		}
// 		hinhAnh = imageURL
// 	}

// 	theTag := c.PostForm("the_tag")
// 	voice := c.DefaultPostForm("voice", "vi-VN-Chirp3-HD-Puck")
// 	speakingRateStr := c.DefaultPostForm("speaking_rate", "1.0")
// 	rateValue, _ := strconv.ParseFloat(speakingRateStr, 64)
// 	if rateValue <= 0 {
// 		rateValue = 1.0
// 	}

// 	authHeader := c.GetHeader("Authorization")
// 	parts := strings.Split(authHeader, " ")
// 	if len(parts) != 2 || parts[0] != "Bearer" {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header không hợp lệ"})
// 		return
// 	}
// 	token := parts[1]

// 	respData, err := services.CallUploadDocumentAPI(file, userID, token, voice, rateValue)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi gọi UploadDocument", "details": err.Error()})
// 		return
// 	}

// 	taiLieuRaw, ok := respData["tai_lieu"]
// 	if !ok {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy dữ liệu tài liệu từ UploadDocument"})
// 		return
// 	}

// 	taiLieuMap, ok := taiLieuRaw.(map[string]interface{})
// 	if !ok {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu tài liệu không đúng định dạng"})
// 		return
// 	}

// 	audioURL, _ := respData["audio_url"].(string)
// 	taiLieuID, _ := taiLieuMap["id"].(string)

// 	durationFloat, _ := services.GetMP3DurationFromURL(audioURL)
// 	totalSeconds := int(durationFloat)

// 	// ✅ Tính toán VIP status ngay khi tạo
// 	isVIP := false
// 	if totalSeconds > 180 {
// 		isVIP = true // Podcast dài > 3 phút
// 	}
// 	// Podcast mới luôn là VIP (trong 7 ngày)
// 	isVIP = true

// 	podcast := models.Podcast{
// 		ID:             uuid.New().String(),
// 		TailieuID:      taiLieuID,
// 		TieuDe:         tieuDe,
// 		MoTa:           moTa,
// 		DuongDanAudio:  audioURL,
// 		ThoiLuongGiay:  totalSeconds,
// 		HinhAnhDaiDien: hinhAnh,
// 		DanhMucID:      danhMucID,
// 		TrangThai:      "Tắt",
// 		NguoiTao:       userID,
// 		TheTag:         theTag,
// 		LuotXem:        0,
// 		IsVIP:          isVIP, // ✅ Set VIP status
// 	}

// 	if err := db.Create(&podcast).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo podcast", "details": err.Error()})
// 		return
// 	}

// 	message := fmt.Sprintf("Admin %s đã tạo podcast: %s", userID, tieuDe)
// 	services.CreateNotification(userID, podcast.ID, "create_podcast", message)

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Tạo podcast thành công",
// 		"podcast": podcast,
// 	})
// }

// // ✅ Modified GetPodcast - Tự động sync VIP status
// func GetPodcast(c *gin.Context) {
// 	db := config.DB
// 	var podcasts []models.Podcast
// 	var total int64

// 	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
// 	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
// 	offset := (page - 1) * limit

// 	search := c.Query("search")
// 	status := c.Query("status")
// 	categoryID := c.Query("category")
// 	sort := c.DefaultQuery("sort", "date")
// 	vipFilter := c.Query("vip") // "true", "false", hoặc ""

// 	query := db.Model(&models.Podcast{}).Preload("TaiLieu").Preload("DanhMuc")

// 	role, _ := c.Get("vai_tro")
// 	if role != "admin" {
// 		query = query.Where("trang_thai = ?", "Bật")
// 	}

// 	if search != "" {
// 		query = query.Where("LOWER(tieu_de) LIKE ?", "%"+strings.ToLower(search)+"%")
// 	}

// 	if categoryID != "" {
// 		query = query.Where("danh_muc_id = ?", categoryID)
// 	}

// 	if status != "" && role == "admin" {
// 		switch status {
// 		case "Bật":
// 			query = query.Where("trang_thai = ?", "Bật")
// 		case "Tắt":
// 			query = query.Where("trang_thai = ?", "Tắt")
// 		}
// 	}

// 	// ✅ Filter VIP từ database
// 	if vipFilter == "true" {
// 		query = query.Where("is_vip = ?", true)
// 	} else if vipFilter == "false" {
// 		query = query.Where("is_vip = ?", false)
// 	}

// 	orderBy := "ngay_tao_ra DESC"
// 	if sort == "views" {
// 		orderBy = "luot_xem DESC"
// 	}

// 	query.Count(&total)
// 	query.Order(orderBy).Offset(offset).Limit(limit).Find(&podcasts)

// 	// ✅ Tự động sync VIP status (chạy background)
// 	go BatchUpdateVIPStatus(db, podcasts)

// 	AttachSummary(db, podcasts)

// 	c.JSON(http.StatusOK, gin.H{
// 		"data": podcasts,
// 		"pagination": gin.H{
// 			"page":        page,
// 			"limit":       limit,
// 			"total":       total,
// 			"total_pages": (total + int64(limit) - 1) / int64(limit),
// 		},
// 	})
// }

// // ✅ Xem chi tiết podcast (WITH VIP CHECK CHỈ CHO USER THƯỜNG)
// func GetPodcastByID(c *gin.Context) {
// 	db := config.DB
// 	id := c.Param("id")
// 	var podcast models.Podcast

// 	if err := db.Preload("TaiLieu").Preload("DanhMuc").First(&podcast, "id = ?", id).Error; err != nil {
// 		if err == gorm.ErrRecordNotFound {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy podcast"})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy thông tin podcast"})
// 		}
// 		return
// 	}

// 	// ✅ Sync VIP status (nếu cần)
// 	UpdatePodcastVIPStatus(db, &podcast)

// 	role, _ := c.Get("vai_tro")

// 	// ✅ ADMIN bỏ qua kiểm tra VIP - có toàn quyền truy cập
// 	if role != "admin" && podcast.IsVIP {
// 		// Chỉ kiểm tra VIP nếu là USER THƯỜNG
// 		userIDStr := c.GetString("user_id")

// 		if userIDStr == "" {
// 			c.JSON(http.StatusForbidden, gin.H{
// 				"error":           "VIP Required",
// 				"message":         "Podcast này yêu cầu VIP. Vui lòng đăng nhập và nâng cấp tài khoản VIP để nghe.",
// 				"is_vip_required": true,
// 				"requires_login":  true,
// 				"podcast_preview": gin.H{
// 					"id":                podcast.ID,
// 					"tieu_de":           podcast.TieuDe,
// 					"mo_ta":             podcast.MoTa,
// 					"hinh_anh_dai_dien": podcast.HinhAnhDaiDien,
// 					"thoi_luong_giay":   podcast.ThoiLuongGiay,
// 					"danh_muc":          podcast.DanhMuc,
// 					"is_vip":            true,
// 				},
// 			})
// 			return
// 		}

// 		var user models.NguoiDung
// 		if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác thực người dùng"})
// 			return
// 		}

// 		if !IsUserVIP(&user) {
// 			vipExpired := user.VIP && user.VIPExpires != nil && time.Now().After(*user.VIPExpires)

// 			c.JSON(http.StatusForbidden, gin.H{
// 				"error":           "VIP Required",
// 				"message":         "Podcast này chỉ dành cho thành viên VIP. Vui lòng nâng cấp tài khoản để tiếp tục.",
// 				"is_vip_required": true,
// 				"vip_expired":     vipExpired,
// 				"podcast_preview": gin.H{
// 					"id":                podcast.ID,
// 					"tieu_de":           podcast.TieuDe,
// 					"mo_ta":             podcast.MoTa,
// 					"hinh_anh_dai_dien": podcast.HinhAnhDaiDien,
// 					"thoi_luong_giay":   podcast.ThoiLuongGiay,
// 					"danh_muc":          podcast.DanhMuc,
// 					"is_vip":            true,
// 				},
// 			})
// 			return
// 		}
// 	}

// 	// ✅ Được phép truy cập - tăng lượt xem
// 	db.Model(&podcast).UpdateColumn("luot_xem", gorm.Expr("luot_xem + ?", 1))

// 	// ✅ AUTO-SAVE LISTENING HISTORY - Fixed to use string IDs
// 	userIDStr := c.GetString("user_id")
// 	if userIDStr != "" {
// 		var history models.LichSuNghe
// 		now := time.Now()

// 		result := db.Where("nguoi_dung_id = ? AND podcast_id = ?", userIDStr, id).First(&history)

// 		if result.Error == gorm.ErrRecordNotFound {
// 			// Tạo mới lịch sử
// 			history = models.LichSuNghe{
// 				ID:          uuid.New().String(),
// 				NguoiDungID: userIDStr,
// 				PodcastID:   id,
// 				ViTri:       0,
// 				NgayNghe:    now,
// 			}
// 			if err := db.Create(&history).Error; err != nil {
// 				// Log error nhưng không block response
// 				fmt.Printf("Lỗi khi lưu lịch sử nghe: %v\n", err)
// 			}
// 		} else if result.Error == nil {
// 			// Cập nhật thời gian nghe
// 			if err := db.Model(&history).Update("ngay_nghe", now).Error; err != nil {
// 				fmt.Printf("Lỗi khi cập nhật lịch sử nghe: %v\n", err)
// 			}
// 		}
// 	}

// 	// Attach summary
// 	if podcast.TailieuID != "" {
// 		podcast.TomTat = podcast.TaiLieu.TomTat
// 	}

// 	// Lấy podcast liên quan
// 	var related []models.Podcast
// 	db.Preload("TaiLieu").Preload("DanhMuc").
// 		Where("danh_muc_id = ? AND id != ?", podcast.DanhMucID, podcast.ID).
// 		Order("ngay_tao_ra DESC").Limit(5).Find(&related)

// 	// Sync VIP cho related
// 	go BatchUpdateVIPStatus(db, related)
// 	AttachSummary(db, related)

// 	c.JSON(http.StatusOK, gin.H{
// 		"data":    podcast,
// 		"suggest": related,
// 	})
// }

// // ✅ Cập nhật podcast - CHỈ ADMIN
// func UpdatePodcast(c *gin.Context) {
// 	role, _ := c.Get("vai_tro")
// 	if role != "admin" {
// 		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền chỉnh sửa podcast"})
// 		return
// 	}

// 	db := c.MustGet("db").(*gorm.DB)
// 	podcastID := c.Param("id")

// 	var podcast models.Podcast
// 	if err := db.First(&podcast, "id = ?", podcastID).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Podcast không tồn tại"})
// 		return
// 	}

// 	tieuDe := c.PostForm("tieu_de")
// 	moTa := c.PostForm("mo_ta")
// 	theTag := c.PostForm("the_tag")
// 	danhMucID := c.PostForm("danh_muc_id")
// 	trangThai := c.PostForm("trang_thai")

// 	// ✅ Admin có thể sửa VIP status
// 	vipStatus := c.PostForm("is_vip") // "true" hoặc "false"

// 	changes := []string{}

// 	if tieuDe != "" && tieuDe != podcast.TieuDe {
// 		changes = append(changes, fmt.Sprintf("tiêu đề: %s → %s", podcast.TieuDe, tieuDe))
// 		podcast.TieuDe = tieuDe
// 	}
// 	if moTa != "" && moTa != podcast.MoTa {
// 		changes = append(changes, "mô tả")
// 		podcast.MoTa = moTa
// 	}
// 	if theTag != "" && theTag != podcast.TheTag {
// 		changes = append(changes, "tag")
// 		podcast.TheTag = theTag
// 	}
// 	if danhMucID != "" && danhMucID != podcast.DanhMucID {
// 		changes = append(changes, "danh mục")
// 		podcast.DanhMucID = danhMucID
// 	}
// 	if trangThai != "" && trangThai != podcast.TrangThai {
// 		changes = append(changes, fmt.Sprintf("trạng thái: %s → %s", podcast.TrangThai, trangThai))
// 		podcast.TrangThai = trangThai
// 		if trangThai == "Bật" {
// 			now := time.Now()
// 			podcast.NgayXuatBan = &now
// 		}
// 	}

// 	// ✅ Xử lý VIP status
// 	if vipStatus != "" {
// 		newVIP := vipStatus == "true"
// 		if newVIP != podcast.IsVIP {
// 			changes = append(changes, fmt.Sprintf("VIP: %v → %v", podcast.IsVIP, newVIP))
// 			podcast.IsVIP = newVIP
// 		}
// 	}

// 	if hinhAnhFile, err := c.FormFile("hinh_anh_dai_dien"); err == nil {
// 		if imageURL, err := utils.UploadImageToSupabase(hinhAnhFile, uuid.New().String()); err == nil {
// 			podcast.HinhAnhDaiDien = imageURL
// 			changes = append(changes, "hình ảnh đại diện")
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể upload hình ảnh"})
// 			return
// 		}
// 	}

// 	if err := db.Save(&podcast).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật podcast"})
// 		return
// 	}

// 	if len(changes) > 0 {
// 		message := fmt.Sprintf("Podcast %s đã được cập nhật: %v", podcast.TieuDe, changes)
// 		services.CreateNotification("", podcast.ID, "update_podcast", message)
// 	}

// 	db.Preload("TaiLieu").Preload("DanhMuc").First(&podcast, "id = ?", podcastID)
// 	AttachSummary(db, []models.Podcast{podcast})

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Cập nhật podcast thành công",
// 		"podcast": podcast,
// 	})
// }

// // ✅ Admin toggle VIP status
// func TogglePodcastVIPStatus(c *gin.Context) {
// 	role, _ := c.Get("vai_tro")
// 	if role != "admin" {
// 		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền thay đổi trạng thái VIP"})
// 		return
// 	}

// 	db := c.MustGet("db").(*gorm.DB)
// 	podcastID := c.Param("id")

// 	var podcast models.Podcast
// 	if err := db.First(&podcast, "id = ?", podcastID).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Podcast không tồn tại"})
// 		return
// 	}

// 	// Toggle VIP status
// 	newVIPStatus := !podcast.IsVIP
// 	if err := db.Model(&podcast).Update("is_vip", newVIPStatus).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái VIP"})
// 		return
// 	}

// 	podcast.IsVIP = newVIPStatus

// 	message := fmt.Sprintf("Admin đã thay đổi trạng thái VIP của podcast '%s' thành: %v", podcast.TieuDe, newVIPStatus)
// 	services.CreateNotification("", podcast.ID, "update_vip", message)

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": fmt.Sprintf("Đã %s trạng thái VIP", map[bool]string{true: "bật", false: "tắt"}[newVIPStatus]),
// 		"podcast": podcast,
// 	})
// }

// // ✅ Sync tất cả VIP status (Admin only - chạy manual)
// func SyncAllVIPStatus(c *gin.Context) {
// 	role, _ := c.Get("vai_tro")
// 	if role != "admin" {
// 		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền sync VIP"})
// 		return
// 	}

// 	db := c.MustGet("db").(*gorm.DB)

// 	var podcasts []models.Podcast
// 	if err := db.Find(&podcasts).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách podcast"})
// 		return
// 	}

// 	updated := 0
// 	for i := range podcasts {
// 		oldStatus := podcasts[i].IsVIP
// 		shouldBeVIP := CheckPodcastVIPStatus(&podcasts[i])

// 		if oldStatus != shouldBeVIP {
// 			db.Model(&podcasts[i]).Update("is_vip", shouldBeVIP)
// 			updated++
// 		}
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message":        "Đã đồng bộ trạng thái VIP",
// 		"total_podcasts": len(podcasts),
// 		"updated":        updated,
// 	})
// }

// // ✅ Các hàm còn lại từ file gốc...
// func SearchPodcast(c *gin.Context) {
// 	db := config.DB
// 	search := c.Query("q")
// 	status := c.Query("trang_thai")

// 	if search == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu từ khoá tìm kiếm"})
// 		return
// 	}

// 	var podcasts []models.Podcast
// 	query := db.Model(&models.Podcast{}).
// 		Where("LOWER(tieu_de) LIKE ? OR LOWER(mo_ta) LIKE ? OR LOWER(the_tag) LIKE ?",
// 			"%"+strings.ToLower(search)+"%",
// 			"%"+strings.ToLower(search)+"%",
// 			"%"+strings.ToLower(search)+"%",
// 		).
// 		Preload("TaiLieu").Preload("DanhMuc")

// 	if status != "" {
// 		query = query.Where("trang_thai = ?", status)
// 	}

// 	if err := query.Find(&podcasts).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tìm kiếm podcast"})
// 		return
// 	}

// 	// ✅ Đánh dấu VIP
// 	for i := range podcasts {
// 		podcasts[i].IsVIP = CheckPodcastVIPStatus(&podcasts[i])
// 	}

// 	AttachSummary(db, podcasts)
// 	c.JSON(http.StatusOK, gin.H{"data": podcasts})
// }

// func GetDisabledPodcasts(c *gin.Context) {
// 	db := config.DB
// 	var podcasts []models.Podcast

// 	if err := db.Where("trang_thai = ?", "Tắt").
// 		Preload("TaiLieu").Preload("DanhMuc").
// 		Order("ngay_tao_ra DESC").Find(&podcasts).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"error":  "Lỗi khi lấy danh sách podcast bị tắt",
// 			"detail": err.Error(),
// 		})
// 		return
// 	}

// 	for i := range podcasts {
// 		podcasts[i].IsVIP = CheckPodcastVIPStatus(&podcasts[i])
// 	}

// 	AttachSummary(db, podcasts)

// 	c.JSON(http.StatusOK, gin.H{
// 		"count": len(podcasts),
// 		"data":  podcasts,
// 	})
// }

// func GetRecommendedPodcasts(c *gin.Context) {
// 	db := config.DB
// 	podcastID := c.Param("id")

// 	var current models.Podcast
// 	if err := db.First(&current, "id = ?", podcastID).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy podcast"})
// 		return
// 	}

// 	type PodcastWithStats struct {
// 		models.Podcast
// 		AvgRating  float64 `json:"avg_rating"`
// 		TotalVotes int64   `json:"total_votes"`
// 		TomTat     string  `json:"tom_tat"`
// 	}

// 	var recommendations []PodcastWithStats

// 	if err := db.Table("podcasts p").
// 		Select(`p.*, COALESCE(AVG(d.sao),0) AS avg_rating, COUNT(d.id) AS total_votes`).
// 		Joins("LEFT JOIN danh_gias d ON d.podcast_id = p.id").
// 		Where("p.danh_muc_id = ? AND p.id != ? AND p.trang_thai = ?", current.DanhMucID, current.ID, "Bật").
// 		Group("p.id").
// 		Order("avg_rating DESC, p.luot_xem DESC, p.ngay_tao_ra DESC").
// 		Limit(6).
// 		Scan(&recommendations).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách đề xuất"})
// 		return
// 	}

// 	for i := range recommendations {
// 		var tl models.TaiLieu
// 		if err := db.First(&tl, "id = ?", recommendations[i].TailieuID).Error; err == nil {
// 			recommendations[i].TomTat = tl.TomTat
// 		}
// 		recommendations[i].IsVIP = CheckPodcastVIPStatus(&recommendations[i].Podcast)
// 	}

// 	if len(recommendations) == 0 {
// 		db.Table("podcasts p").
// 			Select(`p.*, COALESCE(AVG(d.sao),0) AS avg_rating, COUNT(d.id) AS total_votes`).
// 			Joins("LEFT JOIN danh_gias d ON d.podcast_id = p.id").
// 			Where("p.id != ? AND p.trang_thai = ?", current.ID, "Bật").
// 			Group("p.id").
// 			Order("avg_rating DESC, total_votes DESC").
// 			Limit(6).
// 			Scan(&recommendations)

// 		for i := range recommendations {
// 			var tl models.TaiLieu
// 			if err := db.First(&tl, "id = ?", recommendations[i].TailieuID).Error; err == nil {
// 				recommendations[i].TomTat = tl.TomTat
// 			}
// 			recommendations[i].IsVIP = CheckPodcastVIPStatus(&recommendations[i].Podcast)
// 		}
// 	}

// 	c.JSON(http.StatusOK, gin.H{"data": recommendations})
// }

// // ✅ Lấy podcast nổi bật
// func GetFeaturedPodcasts(c *gin.Context) {
// 	db := config.DB
// 	var podcasts []models.Podcast

// 	// Lấy top 10 podcast có lượt xem cao nhất trong 30 ngày gần đây
// 	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

// 	if err := db.Where("trang_thai = ? AND ngay_tao_ra >= ?", "Bật", thirtyDaysAgo).
// 		Preload("TaiLieu").
// 		Preload("DanhMuc").
// 		Order("luot_xem DESC").
// 		Limit(10).
// 		Find(&podcasts).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"error": "Không thể lấy danh sách podcast nổi bật",
// 		})
// 		return
// 	}

// 	// Sync VIP status
// 	for i := range podcasts {
// 		podcasts[i].IsVIP = CheckPodcastVIPStatus(&podcasts[i])
// 	}

// 	AttachSummary(db, podcasts)

// 	c.JSON(http.StatusOK, gin.H{
// 		"data": podcasts,
// 	})
// }
