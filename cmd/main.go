package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv" // ✅ Thêm dòng này để dùng godotenv
)

func main() {
	if os.Getenv("DOCKER_ENV") != "true" {
		_ = godotenv.Load() // chỉ dùng khi chạy local, không lỗi khi thiếu
	}

	// Connect DB
	config.ConnectDB()

	// Setup Gin
	r := gin.Default()

	// ✅ Bổ sung cấu hình CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",            // ✅ React local
			"https://your-frontend-domain.com", // ✅ nếu bạn có deploy
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Setup routes
	routes.SetupRoutes(r, config.DB)

	// Get port from environment (Railway sets PORT automatically)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default local
	}

	fmt.Printf("🚀 Server starting on port %s\n", port)

	// Start server
	log.Fatal(r.Run(":" + port))
}
