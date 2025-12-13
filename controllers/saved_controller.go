package controllers

// import (
// 	"net/http"
// 	"time"

// 	"github.com/Huong3203/APIPodcast/models"
// 	"github.com/gin-gonic/gin"
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// // ==================== TOGGLE SAVED PODCAST ====================
// func ToggleLuuPodcast(c *gin.Context) {
// 	db := c.MustGet("db").(*gorm.DB)

// 	userID, _ := uuid.Parse(c.GetString("user_id"))
// 	podcastID, err := uuid.Parse(c.Param("id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast_id"})
// 		return
// 	}

// 	// Check if podcast exists
// 	var podcast models.Podcast
// 	if err := db.First(&podcast, "id = ?", podcastID).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Podcast not found"})
// 		return
// 	}

// 	var saved models.SavedPodcast
// 	err = db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).First(&saved).Error

// 	if err == gorm.ErrRecordNotFound {
// 		// Add to saved
// 		saved = models.SavedPodcast{
// 			ID:        uuid.New(),
// 			UserID:    userID,
// 			PodcastID: podcastID,
// 			SavedAt:   time.Now(),
// 		}

// 		if err := db.Create(&saved).Error; err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save podcast"})
// 			return
// 		}

// 		// Increment save count
// 		db.Model(&models.Podcast{}).
// 			Where("id = ?", podcastID).
// 			UpdateColumn("save_count", gorm.Expr("save_count + 1"))

// 		c.JSON(http.StatusOK, gin.H{
// 			"message":  "Podcast saved",
// 			"is_saved": true,
// 			"data":     saved,
// 		})
// 		return
// 	}

// 	// Remove from saved
// 	db.Delete(&saved)

// 	// Decrement save count
// 	db.Model(&models.Podcast{}).
// 		Where("id = ?", podcastID).
// 		UpdateColumn("save_count", gorm.Expr("GREATEST(save_count - 1, 0)"))

// 	c.JSON(http.StatusOK, gin.H{
// 		"message":  "Podcast unsaved",
// 		"is_saved": false,
// 	})
// }

// // ==================== GET SAVED PODCASTS ====================
// func GetMySavedPodcasts(c *gin.Context) {
// 	db := c.MustGet("db").(*gorm.DB)

// 	userID, _ := uuid.Parse(c.GetString("user_id"))

// 	var savedList []models.SavedPodcast
// 	err := db.Where("user_id = ?", userID).
// 		Preload("Podcast").
// 		Order("saved_at DESC").
// 		Find(&savedList).Error

// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get saved podcasts"})
// 		return
// 	}

// 	// Extract podcasts
// 	var podcasts []models.Podcast
// 	for _, saved := range savedList {
// 		podcasts = append(podcasts, saved.Podcast)
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"data":  podcasts,
// 		"total": len(podcasts),
// 	})
// }
