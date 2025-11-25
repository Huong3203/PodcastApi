package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/routes"
	"github.com/Huong3203/APIPodcast/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load env khi không chạy Docker
	if os.Getenv("DOCKER_ENV") != "true" {
		_ = godotenv.Load()
	}

	config.ConnectDB()

	// Auto migrate tất cả models ứng dụng
	config.DB.AutoMigrate(
		&models.NguoiDung{},
		&models.Payment{},
		&models.Podcast{},
		&models.DanhGia{},
		&models.FeaturedRating{},
	)

	// WebSocket background worker
	go ws.HandleNotificationMessages()

	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"https://your-frontend-domain.com",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Setup routes
	routes.SetupRoutes(r, config.DB)

	// PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("🚀 Server running on port " + port)
	log.Fatal(r.Run(":" + port))
}
