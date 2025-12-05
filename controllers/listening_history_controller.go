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

// ========================= SAVE LISTENING HISTORY =========================
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

	// Check podcast exist
	var podcast models.Podcast
	if err := db.First(&podcast, "id = ?", podcastID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Podcast not found"})
		return
	}

	var history models.ListeningHistory
	result := db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).First(&history)
	now := time.Now()

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {

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

		// Increase view count only first time
		db.Model(&models.Podcast{}).Where("id = ?", podcastID).
			UpdateColumn("view_count", gorm.Expr("view_count + 1"))

	} else if result.Error == nil {

		history.LastListenedAt = now
		history.LastPosition = req.LastPosition

		// Update duration when higher
		if req.Duration > history.Duration {
			history.Duration = req.Duration
		}

		// Mark completed
		if req.Completed != nil && *req.Completed && !history.Completed {
			history.Completed = true
			history.CompletedAt = &now
		}

		if err := db.Save(&history).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update listening history"})
			return
		}

	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	db.Preload("Podcast").First(&history, "id = ?", history.ID)
	c.JSON(http.StatusOK, gin.H{
		"message": "History saved",
		"data":    history,
	})
}

// ========================= GET LISTENING HISTORY =========================
// GET /api/user/listening-history?completed=&page=&limit=&sort=&time=&from=&to=
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

	// Paging
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Sorting
	sortOrder := c.DefaultQuery("sort", "desc")
	orderClause := "last_listened_at DESC"
	if sortOrder == "asc" {
		orderClause = "last_listened_at ASC"
	}

	query := db.Model(&models.ListeningHistory{}).Where("user_id = ?", userID)

	// Filter by completed
	if completed := c.Query("completed"); completed != "" {
		if completed == "true" {
			query = query.Where("completed = ?", true)
		} else if completed == "false" {
			query = query.Where("completed = ?", false)
		}
	}

	// Time filter
	timeFilter := c.DefaultQuery("time", "all")
	now := time.Now()
	start := time.Time{}

	switch timeFilter {
	case "today":
		start = now.Truncate(24 * time.Hour)
		query = query.Where("last_listened_at >= ?", start)
	case "week":
		start = now.AddDate(0, 0, -7)
		query = query.Where("last_listened_at >= ?", start)
	case "month":
		start = now.AddDate(0, -1, 0)
		query = query.Where("last_listened_at >= ?", start)
	case "year":
		start = now.AddDate(-1, 0, 0)
		query = query.Where("last_listened_at >= ?", start)
	case "custom":
		from := c.Query("from")
		to := c.Query("to")

		if from != "" && to != "" {
			fromTime, err1 := time.Parse("2006-01-02", from)
			toTime, err2 := time.Parse("2006-01-02", to)
			if err1 == nil && err2 == nil {
				query = query.Where("last_listened_at BETWEEN ? AND ?", fromTime, toTime)
			}
		}
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Fetch data
	var history []models.ListeningHistory
	err = query.Preload("Podcast").
		Order(orderClause).
		Limit(limit).
		Offset(offset).
		Find(&history).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       history,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": int(math.Ceil(float64(total) / float64(limit))),
	})
}

// ========================= GET PODCAST HISTORY =========================
// GET /api/user/listening-history/:podcast_id
func GetPodcastHistory(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID, _ := uuid.Parse(userIDStr)

	podcastIDStr := c.Param("podcast_id")
	podcastID, err := uuid.Parse(podcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast id"})
		return
	}

	var history models.ListeningHistory
	err = db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).
		Preload("Podcast").
		First(&history).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": history})
}

// ========================= DELETE PODCAST HISTORY =========================
// DELETE /api/user/listening-history/:podcast_id
func DeletePodcastHistory(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)

	podcastIDStr := c.Param("podcast_id")
	podcastID, err := uuid.Parse(podcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast id"})
		return
	}

	db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).Delete(&models.ListeningHistory{})
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// ========================= CLEAR ALL HISTORY =========================
// DELETE /api/user/listening-history
func ClearAllHistory(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)

	db.Where("user_id = ?", userID).Delete(&models.ListeningHistory{})
	c.JSON(http.StatusOK, gin.H{"message": "All history cleared"})
}
