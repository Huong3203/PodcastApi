// controllers/listening_history_controller.go
package controllers

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SavePodcastHistoryRequest struct {
	LastPosition int   `json:"last_position" binding:"required,min=0"`
	Duration     int   `json:"duration" binding:"required,min=1"`
	Completed    *bool `json:"completed,omitempty"`
}

// SavePodcastHistory - Lưu/cập nhật lịch sử nghe
// POST /api/user/listening-history/:podcast_id
func SavePodcastHistory(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	podcastIDStr := c.Param("podcast_id")
	podcastID, err := uuid.Parse(podcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast_id"})
		return
	}

	var req SavePodcastHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Kiểm tra podcast tồn tại
	var podcast models.Podcast
	if err := db.First(&podcast, "id = ?", podcastID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Podcast not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query podcast"})
		return
	}

	var history models.ListeningHistory
	result := db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).First(&history)
	now := time.Now()

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Lượt nghe đầu tiên
		history = models.ListeningHistory{
			UserID:          userID,
			PodcastID:       podcastID,
			LastPosition:    req.LastPosition,
			Duration:        req.Duration,
			FirstListenedAt: now,
			LastListenedAt:  now,
			Completed:       false,
		}
		if req.Completed != nil && *req.Completed {
			history.Completed = true
			history.CompletedAt = &now
		}

		if err := db.Create(&history).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create listening history"})
			return
		}

		// Tăng view count
		db.Model(&models.Podcast{}).Where("id = ?", podcastID).
			UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))

	} else if result.Error == nil {
		// Đã từng nghe
		history.LastListenedAt = now
		history.LastPosition = req.LastPosition

		if req.Duration > history.Duration {
			history.Duration = req.Duration
		}

		if req.Completed != nil && *req.Completed && !history.Completed {
			history.Completed = true
			history.CompletedAt = &now
		}

		if err := db.Save(&history).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update listening history"})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}

	db.Preload("Podcast").First(&history, "id = ?", history.ID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Listening history saved successfully",
		"data":    history,
	})
}

// GetListeningHistory - Lấy danh sách lịch sử nghe
// GET /api/user/listening-history
func GetListeningHistory(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	// Phân trang
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Sắp xếp
	sortOrder := c.DefaultQuery("sort", "desc")
	orderClause := "last_listened_at DESC"
	if sortOrder == "asc" {
		orderClause = "last_listened_at ASC"
	}

	// Lọc theo completed
	completed := c.Query("completed")

	// Lọc theo thời gian
	timeFilter := c.DefaultQuery("time", "all")
	startDate := time.Time{}
	endDate := time.Now()

	switch timeFilter {
	case "today":
		startDate = time.Now().Truncate(24 * time.Hour)
	case "week":
		startDate = time.Now().AddDate(0, 0, -7)
	case "month":
		startDate = time.Now().AddDate(0, -1, 0)
	case "year":
		startDate = time.Now().AddDate(-1, 0, 0)
	case "custom":
		from := c.Query("from")
		to := c.Query("to")
		if from != "" {
			if parsed, err := time.Parse("2006-01-02", from); err == nil {
				startDate = parsed
			}
		}
		if to != "" {
			if parsed, err := time.Parse("2006-01-02", to); err == nil {
				endDate = parsed
			}
		}
	}

	query := db.Model(&models.ListeningHistory{}).
		Where("user_id = ?", userID).
		Preload("Podcast.Chapter.Subject").
		Preload("Podcast.Categories").
		Preload("Podcast.Tags")

	switch completed {
	case "true":
		query = query.Where("completed = ?", true)
	case "false":
		query = query.Where("completed = ?", false)
	}

	if !startDate.IsZero() {
		query = query.Where("last_listened_at BETWEEN ? AND ?", startDate, endDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot count history"})
		return
	}

	var histories []models.ListeningHistory
	if err := query.Order(orderClause).Limit(limit).Offset(offset).Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot get history"})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"message": "Get listening history successfully",
		"data":    histories,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
		"filters": gin.H{
			"time":      timeFilter,
			"completed": completed,
			"sort":      sortOrder,
		},
	})
}

// GetPodcastHistory - Lấy lịch sử nghe của 1 podcast
// GET /api/user/listening-history/:podcast_id
func GetPodcastHistory(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	podcastIDStr := c.Param("podcast_id")
	podcastID, err := uuid.Parse(podcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast_id"})
		return
	}

	db := c.MustGet("db").(*gorm.DB)

	var history models.ListeningHistory
	if err := db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).
		Preload("Podcast").
		First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "No listening history found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": history})
}

// DeletePodcastHistory - Xóa lịch sử nghe của 1 podcast
// DELETE /api/user/listening-history/:podcast_id
func DeletePodcastHistory(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	podcastIDStr := c.Param("podcast_id")
	podcastID, err := uuid.Parse(podcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast_id"})
		return
	}

	db := c.MustGet("db").(*gorm.DB)

	result := db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).
		Delete(&models.ListeningHistory{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No history found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Listening history deleted successfully"})
}

// ClearAllHistory - Xóa toàn bộ lịch sử nghe
// DELETE /api/user/listening-history
func ClearAllHistory(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	db := c.MustGet("db").(*gorm.DB)

	if err := db.Where("user_id = ?", userID).Delete(&models.ListeningHistory{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All listening history cleared successfully"})
}
