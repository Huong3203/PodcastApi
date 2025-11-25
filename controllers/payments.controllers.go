package controllers

import (
	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Lấy tất cả payment VIP đã thành công, kèm thông tin user
func GetAllVIPPayments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payments []struct {
			UserID string `json:"user_id"`
			Email  string `json:"email"`
			HoTen  string `json:"ho_ten"`
			Status string `json:"status"`
			Amount int    `json:"amount"`
		}

		err := db.Table("payments").
			Select("payments.user_id, nguoi_dungs.email, nguoi_dungs.ho_ten, payments.status, payments.amount").
			Joins("left join nguoi_dungs on payments.user_id = nguoi_dungs.id").
			Where("payments.status = ?", "success").
			Scan(&payments).Error

		if err != nil {
			c.JSON(500, gin.H{"error": "Cannot fetch VIP payments"})
			return
		}

		c.JSON(200, gin.H{"data": payments})
	}
}

// Lấy danh sách user đã mua VIP
func GetVIPUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.NguoiDung
		if err := db.Where("vip = ?", true).Find(&users).Error; err != nil {
			c.JSON(500, gin.H{"error": "Cannot fetch VIP users"})
			return
		}

		c.JSON(200, gin.H{"data": users})
	}
}

func GetUserVIPHistory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("userId")

		var payments []models.Payment
		err := db.Where("user_id = ? AND podcast_id IS NULL", userId).
			Order("created_at DESC").
			Find(&payments).Error

		if err != nil {
			c.JSON(500, gin.H{"error": "Cannot fetch payment history"})
			return
		}

		c.JSON(200, gin.H{"data": payments})
	}
}

func CheckUserVIP(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("userId")

		var user models.NguoiDung
		if err := db.First(&user, "id = ?", userId).Error; err != nil {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}

		c.JSON(200, gin.H{
			"user_id": userId,
			"vip":     user.VIP,
		})
	}
}
