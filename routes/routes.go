package routes

import (
	"github.com/Huong3203/APIPodcast/controllers"
	"github.com/Huong3203/APIPodcast/middleware"
	"github.com/Huong3203/APIPodcast/ws"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(r *gin.Engine, db *gorm.DB) {
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	api := r.Group("/api")

	// ---------------- AUTH ----------------
	auth := api.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
	}

	// ---------------- USER ----------------
	user := api.Group("/users")
	{
		user.Use(middleware.AuthMiddleware())
		user.GET("/profile", controllers.GetProfile)
		user.PUT("/profile", controllers.UpdateProfile)
		user.POST("/change-password", controllers.ChangePassword)
	}

	// ---------------- ADMIN ----------------
	admin := api.Group("/admin")
	{
		admin.Use(middleware.AuthMiddleware(), middleware.DBMiddleware(db))
		admin.POST("/documents/upload", controllers.UploadDocument)
		admin.GET("/documents", controllers.ListDocumentStatus)
		admin.POST("/podcasts", controllers.CreatePodcastWithUpload)
		admin.PUT("/podcasts/:id", controllers.UpdatePodcast)
		admin.GET("/stats", controllers.GetAdminStats)

		// === Ratings stats cho admin ===
		admin.GET("/ratings/stats", controllers.GetAdminRatingsStats)

		// === Lấy tất cả user (chỉ admin) ===
		admin.GET("/users", controllers.GetAllUsers)
	}

	// ---------------- CATEGORY ----------------
	category := api.Group("/categories")
	{
		category.GET("/", controllers.GetDanhMucs)
		category.GET("/:id", controllers.GetDanhMucByID)

		adminCategory := category.Group("/")
		adminCategory.Use(middleware.AuthMiddleware())
		{
			adminCategory.POST("/", controllers.CreateDanhMuc)
			adminCategory.PUT("/:id", controllers.UpdateDanhMuc)
			adminCategory.PATCH("/:id/status", controllers.ToggleDanhMucStatus)
		}
	}

	// ---------------- PODCAST ----------------
	publicPodcast := api.Group("/podcasts")
	{
		publicPodcast.GET("/", controllers.GetPodcast)
		publicPodcast.GET("/search", controllers.SearchPodcast)
		publicPodcast.GET("/:id", controllers.GetPodcastByID)

		// === Ratings public ===
		publicPodcast.GET("/:id/ratings", controllers.GetPodcastRatings)
	}

	protectedPodcast := api.Group("/podcasts")
	{
		protectedPodcast.Use(middleware.AuthMiddleware())
		protectedPodcast.POST("/", controllers.CreatePodcastWithUpload)
		protectedPodcast.PUT("/:id", controllers.UpdatePodcast)

		// === Thêm đánh giá (cần login) ===
		protectedPodcast.POST("/:id/ratings", controllers.AddPodcastRating)
	}

	// ---------------- OTHER ----------------
	r.GET("/health", controllers.HealthCheck)
	r.GET("/ws/document/:id", ws.HandleDocumentWebSocket)
	r.GET("/ws/status", ws.HandleGlobalWebSocket)
}
