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
		auth.POST("/google/clerk", controllers.LoginWithClerk)
	}

	// ---------------- USER ----------------
	user := api.Group("/users")
	{
		user.Use(middleware.AuthMiddleware())
		user.GET("/profile", controllers.GetProfile)
		user.PUT("/profile", controllers.UpdateProfile)
		user.POST("/change-password", controllers.ChangePassword)
	}

	// ---------------- USER NOTIFICATIONS ----------------
	userNotifications := api.Group("/notifications")
	userNotifications.Use(middleware.AuthMiddleware())
	{
		userNotifications.GET("/me", controllers.GetMyNotifications)                 // lấy tất cả thông báo của user
		userNotifications.PUT("/:id/read", controllers.MarkMyNotificationAsRead)     // đánh dấu 1 thông báo đã đọc
		userNotifications.PUT("/read-all", controllers.MarkAllMyNotificationsAsRead) // đánh dấu tất cả đã đọc
	}

	// ---------------- ADMIN ----------------
	admin := api.Group("/admin")
	{
		admin.Use(middleware.AuthMiddleware(), middleware.DBMiddleware(db))

		// Quản lý documents
		admin.POST("/documents/upload", controllers.UploadDocument)
		admin.GET("/documents", controllers.ListDocumentStatus)

		// Quản lý podcasts
		admin.POST("/podcasts", controllers.CreatePodcastWithUpload)
		admin.PUT("/podcasts/:id", controllers.UpdatePodcast)

		// Thống kê
		admin.GET("/stats", controllers.GetAdminStats)
		admin.GET("/ratings/stats", controllers.GetAdminRatingsStats)

		// Quản lý users
		admin.GET("/users", controllers.GetAllUsers)
		admin.PATCH("/users/:id/role", controllers.UpdateUserRole)
		admin.PATCH("/users/:id/toggle-active", controllers.ToggleUserActivation)

		// Quản lý thông báo admin
		admin.GET("/notifications", controllers.GetAllNotifications)             // tất cả
		admin.GET("/notifications/filter", controllers.GetNotificationsByAction) // lọc theo action
		admin.PUT("/notifications/:id/read", controllers.MarkNotificationAsRead) // đánh dấu 1 thông báo đã đọc
		admin.PUT("/notifications/read-all", controllers.MarkAllAsRead)          // đánh dấu tất cả đã đọc
		admin.DELETE("/notifications/:id", controllers.DeleteNotification)       // xóa thông báo
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
		publicPodcast.GET("/disabled", controllers.GetDisabledPodcasts)
		publicPodcast.GET("/:id/ratings", controllers.GetPodcastRatings)

		// ⭐ PODCAST NỔI BẬT
		publicPodcast.GET("/featured", controllers.GetFeaturedPodcasts)

		publicPodcast.GET("/:id/recommendations", controllers.GetRecommendedPodcasts)
	}

	// ---------------- FEATURED RATINGS ----------------
	featuredRatings := api.Group("/ratings")
	{
		// ⭐ ĐÁNH GIÁ NỔI BẬT
		featuredRatings.GET("/featured", controllers.GetFeaturedRatings)
	}

	protectedPodcast := api.Group("/podcasts")
	{
		protectedPodcast.Use(middleware.AuthMiddleware())
		protectedPodcast.POST("/", controllers.CreatePodcastWithUpload)
		protectedPodcast.PUT("/:id", controllers.UpdatePodcast)
		protectedPodcast.POST("/:id/ratings", controllers.AddPodcastRating)

		// YÊU THÍCH
		protectedPodcast.POST("/:id/favorite", controllers.ToggleYeuThichPodcast)
		protectedPodcast.GET("/favorites/me", controllers.GetMyFavoritePodcasts)

		// LƯU THƯ VIỆN
		protectedPodcast.POST("/:id/save", controllers.ToggleLuuPodcast)
		protectedPodcast.GET("/saved/me", controllers.GetMySavedPodcasts)
	}

	// ---------------- OTHER ----------------
	r.GET("/health", controllers.HealthCheck)

	// ---------------- WEBSOCKET ----------------
	r.GET("/ws/document/:id", ws.HandleDocumentWebSocket)
	r.GET("/ws/status", ws.HandleGlobalWebSocket)

	// WebSocket thông báo real-time
	r.GET("/ws/notifications", func(c *gin.Context) {
		ws.HandleNotificationWS(c.Writer, c.Request)
	})

	// Goroutine chạy nền để gửi thông báo đến clients
	go ws.HandleNotificationMessages()
}
