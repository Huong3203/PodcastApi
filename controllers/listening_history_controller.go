package controllers

// import (
// 	"errors"
// 	"math"
// 	"net/http"
// 	"strconv"
// 	"time"

// 	"github.com/Huong3203/APIPodcast/models"
// 	"github.com/gin-gonic/gin"
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// // ========================= SAVE HISTORY (SIMPLE) =========================
// func SavePodcastHistory(c *gin.Context) {
// 	db := c.MustGet("db").(*gorm.DB)

// 	userIDStr := c.GetString("user_id")
// 	if userIDStr == "" {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
// 		return
// 	}
// 	userID, _ := uuid.Parse(userIDStr)

// 	podcastIDStr := c.Param("podcast_id")
// 	podcastID, err := uuid.Parse(podcastIDStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast_id"})
// 		return
// 	}

// 	// Check podcast exist
// 	var podcast models.Podcast
// 	if err := db.First(&podcast, "id = ?", podcastID).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Podcast not found"})
// 		return
// 	}

// 	var history models.ListeningHistory
// 	now := time.Now()

// 	result := db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).First(&history)

// 	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
// 		// Create new history
// 		history = models.ListeningHistory{
// 			ID:         uuid.New(),
// 			UserID:     userID,
// 			PodcastID:  podcastID,
// 			ListenedAt: now,
// 		}

// 		if err := db.Create(&history).Error; err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create history"})
// 			return
// 		}

// 		// Increase view count first time only
// 		db.Model(&models.Podcast{}).
// 			Where("id = ?", podcastID).
// 			UpdateColumn("view_count", gorm.Expr("view_count + 1"))

// 	} else if result.Error == nil {
// 		// Just update listened time
// 		history.ListenedAt = now
// 		if err := db.Save(&history).Error; err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
// 			return
// 		}
// 	} else {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
// 		return
// 	}

// 	db.Preload("Podcast").First(&history, "id = ?", history.ID)
// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "History saved",
// 		"data":    history,
// 	})
// }

// // ========================= GET HISTORY LIST =========================
// func GetListeningHistory(c *gin.Context) {
// 	db := c.MustGet("db").(*gorm.DB)

// 	userID, _ := uuid.Parse(c.GetString("user_id"))

// 	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
// 	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
// 	offset := (page - 1) * limit

// 	sortOrder := c.DefaultQuery("sort", "desc")
// 	orderClause := "listened_at DESC"
// 	if sortOrder == "asc" {
// 		orderClause = "listened_at ASC"
// 	}

// 	query := db.Model(&models.ListeningHistory{}).Where("user_id = ?", userID)

// 	// Time filter
// 	timeFilter := c.DefaultQuery("time", "all")
// 	now := time.Now()
// 	var start time.Time

// 	switch timeFilter {
// 	case "today":
// 		start = now.Truncate(24 * time.Hour)
// 		query = query.Where("listened_at >= ?", start)
// 	case "week":
// 		start = now.AddDate(0, 0, -7)
// 		query = query.Where("listened_at >= ?", start)
// 	case "month":
// 		start = now.AddDate(0, -1, 0)
// 		query = query.Where("listened_at >= ?", start)
// 	case "year":
// 		start = now.AddDate(-1, 0, 0)
// 		query = query.Where("listened_at >= ?", start)
// 	case "custom":
// 		from := c.Query("from")
// 		to := c.Query("to")
// 		if from != "" && to != "" {
// 			fromTime, _ := time.Parse("2006-01-02", from)
// 			toTime, _ := time.Parse("2006-01-02", to)
// 			query = query.Where("listened_at BETWEEN ? AND ?", fromTime, toTime)
// 		}
// 	}

// 	var total int64
// 	query.Count(&total)

// 	var history []models.ListeningHistory
// 	query.Preload("Podcast").
// 		Order(orderClause).
// 		Limit(limit).
// 		Offset(offset).
// 		Find(&history)

// 	c.JSON(http.StatusOK, gin.H{
// 		"data":       history,
// 		"page":       page,
// 		"limit":      limit,
// 		"total":      total,
// 		"totalPages": int(math.Ceil(float64(total) / float64(limit))),
// 	})
// }

// // ========================= GET ONE PODCAST HISTORY =========================
// func GetPodcastHistory(c *gin.Context) {
// 	db := c.MustGet("db").(*gorm.DB)

// 	userID, _ := uuid.Parse(c.GetString("user_id"))
// 	podcastID, err := uuid.Parse(c.Param("podcast_id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast id"})
// 		return
// 	}

// 	var history models.ListeningHistory
// 	err = db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).
// 		Preload("Podcast").
// 		First(&history).Error

// 	if errors.Is(err, gorm.ErrRecordNotFound) {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"data": history})
// }

// // ========================= DELETE ONE =========================
// func DeletePodcastHistory(c *gin.Context) {
// 	db := c.MustGet("db").(*gorm.DB)

// 	userID, _ := uuid.Parse(c.GetString("user_id"))
// 	podcastID, _ := uuid.Parse(c.Param("podcast_id"))

// 	result := db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).
// 		Delete(&models.ListeningHistory{})

// 	if result.RowsAffected == 0 {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "History not found"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
// }

// // ========================= CLEAR ALL =========================
// func ClearAllHistory(c *gin.Context) {
// 	db := c.MustGet("db").(*gorm.DB)

// 	userID, _ := uuid.Parse(c.GetString("user_id"))

// 	result := db.Where("user_id = ?", userID).Delete(&models.ListeningHistory{})

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "All history cleared",
// 		"deleted": result.RowsAffected,
// 	})
// }
