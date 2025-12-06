package controllers

import (
	"net/http"
	"time"

	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== ADD FAVORITE ====================
func AddFavorite(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, _ := uuid.Parse(c.GetString("user_id"))
	podcastID, err := uuid.Parse(c.Param("podcast_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast_id"})
		return
	}

	// Check if podcast exists
	var podcast models.Podcast
	if err := db.First(&podcast, "id = ?", podcastID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Podcast not found"})
		return
	}

	// Check if already favorited
	var existing models.Favorite
	err = db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Already favorited"})
		return
	}

	// Create favorite
	favorite := models.Favorite{
		ID:        uuid.New(),
		UserID:    userID,
		PodcastID: podcastID,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&favorite).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add favorite"})
		return
	}

	// Increment favorite count
	db.Model(&models.Podcast{}).
		Where("id = ?", podcastID).
		UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1"))

	c.JSON(http.StatusOK, gin.H{
		"message": "Added to favorites",
		"data":    favorite,
	})
}

// ==================== REMOVE FAVORITE ====================
func RemoveFavorite(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, _ := uuid.Parse(c.GetString("user_id"))
	podcastID, err := uuid.Parse(c.Param("podcast_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast_id"})
		return
	}

	result := db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).
		Delete(&models.Favorite{})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Favorite not found"})
		return
	}

	// Decrement favorite count
	db.Model(&models.Podcast{}).
		Where("id = ?", podcastID).
		UpdateColumn("favorite_count", gorm.Expr("GREATEST(favorite_count - 1, 0)"))

	c.JSON(http.StatusOK, gin.H{"message": "Removed from favorites"})
}

// ==================== CHECK FAVORITE ====================
func CheckFavorite(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, _ := uuid.Parse(c.GetString("user_id"))
	podcastID, err := uuid.Parse(c.Param("podcast_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid podcast_id"})
		return
	}

	var favorite models.Favorite
	err = db.Where("user_id = ? AND podcast_id = ?", userID, podcastID).First(&favorite).Error

	isFavorited := err == nil

	c.JSON(http.StatusOK, gin.H{
		"is_favorited": isFavorited,
	})
}

// ==================== GET FAVORITES LIST ====================
func GetFavorites(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, _ := uuid.Parse(c.GetString("user_id"))

	var favorites []models.Favorite
	err := db.Where("user_id = ?", userID).
		Preload("Podcast").
		Order("created_at DESC").
		Find(&favorites).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get favorites"})
		return
	}

	// Extract podcasts
	var podcasts []models.Podcast
	for _, fav := range favorites {
		podcasts = append(podcasts, fav.Podcast)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  podcasts,
		"total": len(podcasts),
	})
}
