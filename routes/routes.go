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
		auth.POST("/google/clerk", controllers.ClerkLogin)
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

		admin.GET("/ratings/stats", controllers.GetAdminRatingsStats)

		admin.GET("/users", controllers.GetAllUsers)
		admin.PATCH("/users/:id/role", controllers.UpdateUserRole)
		admin.PATCH("/users/:id/toggle-active", controllers.ToggleUserActivation)
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

		publicPodcast.GET("/:id/ratings", controllers.GetPodcastRatings)
	}

	protectedPodcast := api.Group("/podcasts")
	{
		protectedPodcast.Use(middleware.AuthMiddleware())
		protectedPodcast.POST("/", controllers.CreatePodcastWithUpload)
		protectedPodcast.PUT("/:id", controllers.UpdatePodcast)

		protectedPodcast.POST("/:id/ratings", controllers.AddPodcastRating)

		// ✅ YÊU THÍCH
		protectedPodcast.POST("/:id/favorite", controllers.ToggleYeuThichPodcast)
		protectedPodcast.GET("/favorites/me", controllers.GetMyFavoritePodcasts)

		// ✅ LƯU THƯ VIỆN
		protectedPodcast.POST("/:id/save", controllers.ToggleLuuPodcast)
		protectedPodcast.GET("/saved/me", controllers.GetMySavedPodcasts)
	}

	// ---------------- OTHER ----------------
	r.GET("/health", controllers.HealthCheck)
	r.GET("/ws/document/:id", ws.HandleDocumentWebSocket)
	r.GET("/ws/status", ws.HandleGlobalWebSocket)
}
