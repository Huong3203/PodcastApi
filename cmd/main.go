package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/controllers"
	"github.com/Huong3203/APIPodcast/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	//Load .env khi chạy local
	if os.Getenv("DOCKER_ENV") != "true" {
		_ = godotenv.Load()
	}

	// Connect MySQL
	config.ConnectDB()

	//Init Clerk Secret Key
	controllers.InitClerk()

	//Setup Gin
	r := gin.Default()

	//CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",            // FE local
			"https://your-frontend-domain.com", // FE deployed
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Routes
	routes.SetupRoutes(r, config.DB)

	// Lấy PORT từ ENV (Railway tự set)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 Server running on port %s\n", port)

	//Start server
	log.Fatal(r.Run(":" + port))
}
