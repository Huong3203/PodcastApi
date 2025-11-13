package controllers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/services"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/Huong3203/APIPodcast/ws"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ================== ADMIN / UPLOAD DOCUMENT (YÊU CẦU ĐĂNG NHẬP) ==================
func UploadDocument(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.GetString("user_id")

	// ✅ BẮT BUỘC ĐĂNG NHẬP
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Vui lòng đăng nhập để tải lên tài liệu"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có file đính kèm"})
		return
	}
	if file.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File vượt quá 20MB"})
		return
	}

	ext := filepath.Ext(file.Filename)
	inputType, err := services.GetInputTypeFromExt(ext)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New().String()
	ws.SendStatusUpdate(id, "Đang tải lên tài liệu...", 0, "")

	publicURL, err := utils.UploadFileToSupabase(file, id)
	if err != nil {
		ws.SendStatusUpdate(id, "Lỗi khi tải lên Supabase", 0, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi upload Supabase", "details": err.Error()})
		return
	}

	doc := models.TaiLieu{
		ID:            id,
		TenFileGoc:    file.Filename,
		DuongDanFile:  publicURL,
		LoaiFile:      strings.TrimPrefix(ext, "."),
		KichThuocFile: file.Size,
		TrangThai:     "Đã tải lên",
		NguoiTaiLen:   userID,
	}

	if err := db.Create(&doc).Error; err != nil {
		ws.SendStatusUpdate(id, "Không thể lưu tài liệu vào database", 0, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lưu được tài liệu", "details": err.Error()})
		return
	}

	ws.SendStatusUpdate(id, "Đã tải lên", 10, "")
	ws.BroadcastDocumentListChanged()

	ws.SendStatusUpdate(id, "Đang trích xuất nội dung...", 20, "")
	noiDung, err := services.NormalizeInput(services.InputSource{
		Type:       inputType,
		FileHeader: file,
	})
	if err != nil {
		ws.SendStatusUpdate(id, "Lỗi khi trích xuất nội dung", 0, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trích xuất nội dung", "details": err.Error()})
		return
	}

	ws.SendStatusUpdate(id, "Đang làm sạch nội dung...", 30, "")
	cleanedContent, err := services.CleanTextPipeline(noiDung)
	if err != nil {
		ws.SendStatusUpdate(id, "Lỗi khi làm sạch nội dung", 0, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể làm sạch nội dung", "details": err.Error()})
		return
	}

	fmt.Println("Nội dung đã làm sạch:", cleanedContent)

	db.Model(&doc).Updates(map[string]interface{}{
		"TrangThai":        "Đã trích xuất",
		"NoiDungTrichXuat": cleanedContent,
	})
	ws.SendStatusUpdate(id, "Đã trích xuất", 40, "")
	ws.BroadcastDocumentListChanged()

	ws.SendStatusUpdate(id, "Đang tạo audio...", 50, "")
	voice := c.PostForm("voice")
	if voice == "" {
		voice = "vi-VN-Chirp3-HD-Puck"
	}
	rate := 1.0
	if rateStr := c.PostForm("speaking_rate"); rateStr != "" {
		if parsed, err := strconv.ParseFloat(rateStr, 64); err == nil && parsed > 0 {
			rate = parsed
		}
	}

	audioData, err := services.SynthesizeText(cleanedContent, voice, rate)
	if err != nil {
		ws.SendStatusUpdate(id, "Lỗi khi tạo audio", 0, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo audio", "details": err.Error()})
		return
	}

	ws.SendStatusUpdate(id, "Đang lưu audio...", 60, "")
	audioURL, err := utils.UploadBytesToSupabase(audioData, id+".mp3", "audio/mp3")
	if err != nil {
		ws.SendStatusUpdate(id, "Lỗi upload audio", 0, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể upload audio", "details": err.Error()})
		return
	}

	ws.SendStatusUpdate(id, "Đã lưu audio", 70, "")
	now := time.Now()
	db.Model(&doc).Updates(map[string]interface{}{
		"TrangThai":    "Hoàn thành",
		"NgayXuLyXong": &now,
	})

	// 🔹 Tạo thông báo realtime
	message := fmt.Sprintf("Người dùng %s đã tải lên tài liệu: %s", userID, doc.TenFileGoc)
	if err := services.CreateNotification(userID, doc.ID, "upload_document", message); err != nil {
		fmt.Println("Lỗi khi tạo thông báo:", err)
	}

	ws.SendStatusUpdate(id, "Hoàn thành", 100, "")
	ws.BroadcastDocumentListChanged()

	db.Preload("NguoiDung").First(&doc, "id = ?", doc.ID)
	c.JSON(http.StatusOK, gin.H{
		"message":   "Tải lên thành công",
		"tai_lieu":  doc,
		"audio_url": audioURL,
	})
}

//
// ================== PUBLIC / XEM DANH SÁCH PODCAST (KHÔNG CẦN ĐĂNG NHẬP) ==================
//

type TaiLieuStatusDTO struct {
	ID         string `json:"id"`
	TenFileGoc string `json:"ten_file_goc"`
	TrangThai  string `json:"trang_thai"`
	NgayTaiLen string `json:"ngay_tai_len"`
}

func ListDocumentStatus(c *gin.Context) {
	var taiLieus []models.TaiLieu
	var result []TaiLieuStatusDTO
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	search := c.Query("search")
	query := config.DB.Model(&models.TaiLieu{})

	if search != "" {
		query = query.Where("LOWER(ten_file_goc) LIKE ?", "%"+strings.ToLower(search)+"%")
	}

	// ✅ CHỈ LẤY NHỮNG TÀI LIỆU HOÀN THÀNH (CÔNG KHAI)
	query = query.Where("trang_thai = ?", "Hoàn thành")

	query.Count(&total)

	if err := query.Offset(offset).Limit(limit).Order("ngay_tai_len desc").Find(&taiLieus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách tài liệu", "details": err.Error()})
		return
	}

	for _, doc := range taiLieus {
		result = append(result, TaiLieuStatusDTO{
			ID:         doc.ID,
			TenFileGoc: doc.TenFileGoc,
			TrangThai:  doc.TrangThai,
			NgayTaiLen: doc.NgayTaiLen.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}
