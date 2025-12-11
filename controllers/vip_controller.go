// package controllers

// import (
// 	"net/http"
// 	"strconv"
// 	"time"

// 	"github.com/Huong3203/APIPodcast/config"
// 	"github.com/Huong3203/APIPodcast/models"
// 	"github.com/gin-gonic/gin"
// 	"gorm.io/gorm"
// )

// // ✅ Kiểm tra trạng thái VIP của user hiện tại
// func GetUserVIPStatus(c *gin.Context) {
// 	db := config.DB
// 	userIDStr := c.GetString("user_id")

// 	var user models.NguoiDung
// 	if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
// 		return
// 	}

// 	isVIPActive := false
// 	daysRemaining := 0
// 	expired := false

// 	if user.VIP {
// 		if user.VIPExpires != nil {
// 			timeRemaining := time.Until(*user.VIPExpires)
// 			if timeRemaining > 0 {
// 				isVIPActive = true
// 				daysRemaining = int(timeRemaining.Hours() / 24)
// 			} else {
// 				expired = true
// 			}
// 		} else {
// 			// VIP vĩnh viễn (không có ngày hết hạn)
// 			isVIPActive = true
// 			daysRemaining = -1 // -1 = unlimited
// 		}
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"is_vip":         user.VIP,
// 		"is_vip_active":  isVIPActive,
// 		"vip_expires":    user.VIPExpires,
// 		"days_remaining": daysRemaining,
// 		"expired":        expired,
// 		"auto_renew":     user.AutoRenew,
// 		"user": gin.H{
// 			"id":     user.ID,
// 			"email":  user.Email,
// 			"ho_ten": user.HoTen,
// 		},
// 	})
// }

// // ✅ Lấy danh sách podcast VIP (có phân trang)
// func GetVIPPodcasts(c *gin.Context) {
// 	db := config.DB

// 	page := 1
// 	limit := 20
// 	if p, ok := c.GetQuery("page"); ok {
// 		if pInt, err := strconv.Atoi(p); err == nil && pInt > 0 {
// 			page = pInt
// 		}
// 	}
// 	if l, ok := c.GetQuery("limit"); ok {
// 		if lInt, err := strconv.Atoi(l); err == nil && lInt > 0 && lInt <= 100 {
// 			limit = lInt
// 		}
// 	}
// 	offset := (page - 1) * limit

// 	// Lấy tất cả podcast đang bật
// 	var allPodcasts []models.Podcast
// 	db.Where("trang_thai = ?", "Bật").
// 		Preload("TaiLieu").
// 		Preload("DanhMuc").
// 		Order("ngay_tao_ra DESC").
// 		Find(&allPodcasts)

// 	// Lọc podcast VIP
// 	vipPodcasts := []models.Podcast{}
// 	for i := range allPodcasts {
// 		if CheckPodcastVIPStatus(&allPodcasts[i]) {
// 			allPodcasts[i].IsVIP = true
// 			vipPodcasts = append(vipPodcasts, allPodcasts[i])
// 		}
// 	}

// 	// Phân trang
// 	total := len(vipPodcasts)
// 	start := offset
// 	end := offset + limit
// 	if start > total {
// 		start = total
// 	}
// 	if end > total {
// 		end = total
// 	}

// 	paginatedPodcasts := []models.Podcast{}
// 	if start < total {
// 		paginatedPodcasts = vipPodcasts[start:end]
// 	}

// 	AttachSummary(db, paginatedPodcasts)

// 	c.JSON(http.StatusOK, gin.H{
// 		"data": paginatedPodcasts,
// 		"pagination": gin.H{
// 			"page":        page,
// 			"limit":       limit,
// 			"total":       total,
// 			"total_pages": (total + limit - 1) / limit,
// 		},
// 	})
// }

// // ✅ Kiểm tra xem podcast có yêu cầu VIP không (trước khi play)
// func CheckPodcastVIPRequirement(c *gin.Context) {
// 	db := config.DB
// 	podcastID := c.Param("id")

// 	var podcast models.Podcast
// 	if err := db.Preload("DanhMuc").First(&podcast, "id = ?", podcastID).Error; err != nil {
// 		if err == gorm.ErrRecordNotFound {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy podcast"})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi server"})
// 		}
// 		return
// 	}

// 	// Kiểm tra trạng thái VIP
// 	isVIP := CheckPodcastVIPStatus(&podcast)

// 	response := gin.H{
// 		"podcast_id":     podcast.ID,
// 		"podcast_title":  podcast.TieuDe,
// 		"is_vip":         isVIP,
// 		"can_access":     false,
// 		"requires_login": false,
// 		"requires_vip":   false,
// 	}

// 	if !isVIP {
// 		// Podcast miễn phí - ai cũng xem được
// 		response["can_access"] = true
// 		response["message"] = "Podcast miễn phí, bạn có thể nghe ngay"
// 		c.JSON(http.StatusOK, response)
// 		return
// 	}

// 	// Podcast VIP - kiểm tra user
// 	userIDStr := c.GetString("user_id")
// 	if userIDStr == "" {
// 		// Chưa đăng nhập
// 		response["requires_login"] = true
// 		response["requires_vip"] = true
// 		response["message"] = "Vui lòng đăng nhập để nghe podcast VIP"

// 		// Thêm thông tin lý do VIP
// 		if time.Since(podcast.NgayTaoRa) <= 7*24*time.Hour {
// 			response["vip_reason"] = "Podcast mới (trong vòng 7 ngày)"
// 		} else if podcast.ThoiLuongGiay > 180 {
// 			response["vip_reason"] = "Podcast dài (trên 3 phút)"
// 		}

// 		c.JSON(http.StatusOK, response)
// 		return
// 	}

// 	// Kiểm tra VIP của user
// 	var user models.NguoiDung
// 	if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác thực"})
// 		return
// 	}

// 	if IsUserVIP(&user) {
// 		// User có VIP hợp lệ
// 		response["can_access"] = true
// 		response["message"] = "Bạn có quyền truy cập podcast này"

// 		// Thêm thông tin VIP của user
// 		if user.VIPExpires != nil {
// 			daysRemaining := int(time.Until(*user.VIPExpires).Hours() / 24)
// 			response["vip_days_remaining"] = daysRemaining
// 		}

// 		c.JSON(http.StatusOK, response)
// 		return
// 	}

// 	// User không có VIP hoặc đã hết hạn
// 	response["requires_vip"] = true

// 	if user.VIP && user.VIPExpires != nil && time.Now().After(*user.VIPExpires) {
// 		response["message"] = "VIP của bạn đã hết hạn. Vui lòng gia hạn để tiếp tục"
// 		response["vip_expired"] = true
// 	} else {
// 		response["message"] = "Podcast này yêu cầu tài khoản VIP"
// 	}

// 	// Thêm thông tin lý do VIP
// 	if time.Since(podcast.NgayTaoRa) <= 7*24*time.Hour {
// 		response["vip_reason"] = "Podcast mới (trong vòng 7 ngày)"
// 		daysOld := int(time.Since(podcast.NgayTaoRa).Hours() / 24)
// 		response["podcast_age_days"] = daysOld
// 	} else if podcast.ThoiLuongGiay > 180 {
// 		response["vip_reason"] = "Podcast dài (trên 3 phút)"
// 		response["podcast_duration"] = podcast.ThoiLuongGiay
// 	}

//		c.JSON(http.StatusOK, response)
//	}
package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ✅ Kiểm tra trạng thái VIP của user hiện tại
func GetUserVIPStatus(c *gin.Context) {
	db := config.DB
	userIDStr := c.GetString("user_id")

	var user models.NguoiDung
	if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}

	isVIPActive := false
	daysRemaining := 0
	expired := false

	if user.VIP {
		if user.VIPExpires != nil {
			timeRemaining := time.Until(*user.VIPExpires)
			if timeRemaining > 0 {
				isVIPActive = true
				daysRemaining = int(timeRemaining.Hours() / 24)
			} else {
				expired = true
			}
		} else {
			// VIP vĩnh viễn (không có ngày hết hạn)
			isVIPActive = true
			daysRemaining = -1 // -1 = unlimited
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"is_vip":         user.VIP,
		"is_vip_active":  isVIPActive,
		"vip_expires":    user.VIPExpires,
		"days_remaining": daysRemaining,
		"expired":        expired,
		"auto_renew":     user.AutoRenew,
		"user": gin.H{
			"id":     user.ID,
			"email":  user.Email,
			"ho_ten": user.HoTen,
		},
	})
}

// ✅ Lấy danh sách podcast VIP (có phân trang)
func GetVIPPodcasts(c *gin.Context) {
	db := config.DB

	page := 1
	limit := 20
	if p, ok := c.GetQuery("page"); ok {
		if pInt, err := strconv.Atoi(p); err == nil && pInt > 0 {
			page = pInt
		}
	}
	if l, ok := c.GetQuery("limit"); ok {
		if lInt, err := strconv.Atoi(l); err == nil && lInt > 0 && lInt <= 100 {
			limit = lInt
		}
	}
	offset := (page - 1) * limit

	// Lấy tất cả podcast đang bật
	var allPodcasts []models.Podcast
	db.Where("trang_thai = ?", "Bật").
		Preload("TaiLieu").
		Preload("DanhMuc").
		Order("ngay_tao_ra DESC").
		Find(&allPodcasts)

	// Lọc podcast VIP
	vipPodcasts := []models.Podcast{}
	for i := range allPodcasts {
		if CheckPodcastVIPStatus(&allPodcasts[i]) {
			allPodcasts[i].IsVIP = true
			vipPodcasts = append(vipPodcasts, allPodcasts[i])
		}
	}

	// Phân trang
	total := len(vipPodcasts)
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedPodcasts := []models.Podcast{}
	if start < total {
		paginatedPodcasts = vipPodcasts[start:end]
	}

	AttachSummary(db, paginatedPodcasts)

	c.JSON(http.StatusOK, gin.H{
		"data": paginatedPodcasts,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

// ✅ Kiểm tra xem podcast có yêu cầu VIP không (trước khi play)
func CheckPodcastVIPRequirement(c *gin.Context) {
	db := config.DB
	podcastID := c.Param("id")

	var podcast models.Podcast
	if err := db.Preload("DanhMuc").First(&podcast, "id = ?", podcastID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy podcast"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi server"})
		}
		return
	}

	// Kiểm tra trạng thái VIP
	isVIP := CheckPodcastVIPStatus(&podcast)

	response := gin.H{
		"podcast_id":     podcast.ID,
		"podcast_title":  podcast.TieuDe,
		"is_vip":         isVIP,
		"can_access":     false,
		"requires_login": false,
		"requires_vip":   false,
	}

	// ✅ ADMIN luôn được truy cập
	role, _ := c.Get("vai_tro")
	if role == "admin" {
		response["can_access"] = true
		response["message"] = "Admin có toàn quyền truy cập"
		c.JSON(http.StatusOK, response)
		return
	}

	if !isVIP {
		// Podcast miễn phí - ai cũng xem được
		response["can_access"] = true
		response["message"] = "Podcast miễn phí, bạn có thể nghe ngay"
		c.JSON(http.StatusOK, response)
		return
	}

	// Podcast VIP - kiểm tra user
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		// Chưa đăng nhập
		response["requires_login"] = true
		response["requires_vip"] = true
		response["message"] = "Vui lòng đăng nhập để nghe podcast VIP"

		// Thêm thông tin lý do VIP
		if time.Since(podcast.NgayTaoRa) <= 7*24*time.Hour {
			response["vip_reason"] = "Podcast mới (trong vòng 7 ngày)"
		} else if podcast.ThoiLuongGiay > 180 {
			response["vip_reason"] = "Podcast dài (trên 3 phút)"
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// Kiểm tra VIP của user
	var user models.NguoiDung
	if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác thực"})
		return
	}

	if IsUserVIP(&user) {
		// User có VIP hợp lệ
		response["can_access"] = true
		response["message"] = "Bạn có quyền truy cập podcast này"

		// Thêm thông tin VIP của user
		if user.VIPExpires != nil {
			daysRemaining := int(time.Until(*user.VIPExpires).Hours() / 24)
			response["vip_days_remaining"] = daysRemaining
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// User không có VIP hoặc đã hết hạn
	response["requires_vip"] = true

	if user.VIP && user.VIPExpires != nil && time.Now().After(*user.VIPExpires) {
		response["message"] = "VIP của bạn đã hết hạn. Vui lòng gia hạn để tiếp tục"
		response["vip_expired"] = true
	} else {
		response["message"] = "Podcast này yêu cầu tài khoản VIP"
	}

	// Thêm thông tin lý do VIP
	if time.Since(podcast.NgayTaoRa) <= 7*24*time.Hour {
		response["vip_reason"] = "Podcast mới (trong vòng 7 ngày)"
		daysOld := int(time.Since(podcast.NgayTaoRa).Hours() / 24)
		response["podcast_age_days"] = daysOld
	} else if podcast.ThoiLuongGiay > 180 {
		response["vip_reason"] = "Podcast dài (trên 3 phút)"
		response["podcast_duration"] = podcast.ThoiLuongGiay
	}

	c.JSON(http.StatusOK, response)
}
