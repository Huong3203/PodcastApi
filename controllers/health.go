package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"

	"github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
	// Kiểm tra kết nối với DB
	sqlDB, err := config.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "DB not available", "error": err.Error()})
		return
	}
	err = sqlDB.Ping()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "DB not responding", "error": err.Error()})
		return
	}

	// Trả về OK nếu mọi thứ ổn
	c.JSON(http.StatusOK, gin.H{"status": "OK", "message": "Service is healthy"})
}
