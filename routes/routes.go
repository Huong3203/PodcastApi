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

		// Route cho Google login qua Clerk
		auth.POST("/google/clerk", controllers.ClerkLogin)
	}

	// ---------------- MOMO VIP ----------------
	momo := api.Group("/momo")
	{
		momo.POST("/vip/create", controllers.CreateMomoVIPPayment(db))
		momo.POST("/vip/ipn", controllers.MomoVIPIPN(db))
		momo.GET("/vip/return", controllers.MomoVIPReturn(db))
		momo.GET("/vip/history/:userId", controllers.GetUserVIPHistory(db))
		momo.GET("/vip/status/:userId", controllers.CheckUserVIP(db))
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
		userNotifications.GET("/me", controllers.GetMyNotifications)
		userNotifications.PUT("/:id/read", controllers.MarkMyNotificationAsRead)
		userNotifications.PUT("/read-all", controllers.MarkAllMyNotificationsAsRead)
	}

	// ---------------- ADMIN ----------------
	admin := api.Group("/admin")
	{
		admin.Use(middleware.AuthMiddleware(), middleware.DBMiddleware(db))

		admin.GET("/vip-payments", controllers.GetAllVIPPayments(db))
		admin.GET("/vip-users", controllers.GetVIPUsers(db))

		admin.POST("/documents/upload", controllers.UploadDocument)
		admin.GET("/documents", controllers.ListDocumentStatus)

		admin.POST("/podcasts", controllers.CreatePodcastWithUpload)
		admin.PUT("/podcasts/:id", controllers.UpdatePodcast)

		admin.GET("/stats", controllers.GetAdminStats)
		admin.GET("/ratings/stats", controllers.GetAdminRatingsStats)

		admin.GET("/users", controllers.GetAllUsers)
		admin.PATCH("/users/:id/role", controllers.UpdateUserRole)
		admin.PATCH("/users/:id/toggle-active", controllers.ToggleUserActivation)

		admin.GET("/notifications", controllers.GetAllNotifications)
		admin.GET("/notifications/filter", controllers.GetNotificationsByAction)
		admin.PUT("/notifications/:id/read", controllers.MarkNotificationAsRead)
		admin.PUT("/notifications/read-all", controllers.MarkAllAsRead)
		admin.DELETE("/notifications/:id", controllers.DeleteNotification)
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

		publicPodcast.GET("/featured", controllers.GetFeaturedPodcasts)
		publicPodcast.GET("/:id/recommendations", controllers.GetRecommendedPodcasts)
	}

	// ---------------- FEATURED RATINGS ----------------
	featuredRatings := api.Group("/ratings")
	{
		featuredRatings.GET("/featured", controllers.GetFeaturedReviews)
	}

	protectedPodcast := api.Group("/podcasts")
	{
		protectedPodcast.Use(middleware.AuthMiddleware())
		protectedPodcast.POST("/", controllers.CreatePodcastWithUpload)
		protectedPodcast.PUT("/:id", controllers.UpdatePodcast)
		protectedPodcast.POST("/:id/ratings", controllers.AddPodcastRating)

		protectedPodcast.POST("/:id/favorite", controllers.ToggleYeuThichPodcast)
		protectedPodcast.GET("/favorites/me", controllers.GetMyFavoritePodcasts)

		protectedPodcast.POST("/:id/save", controllers.ToggleLuuPodcast)
		protectedPodcast.GET("/saved/me", controllers.GetMySavedPodcasts)

		protectedPodcast.POST("/:id/history", controllers.LuuLichSuNghe)
		protectedPodcast.GET("/history/me", controllers.GetMyListeningHistory)
	}

	// ---------------- OTHER ----------------
	r.GET("/health", controllers.HealthCheck)

	// ---------------- WebSockets ----------------
	r.GET("/ws/document/:id", ws.HandleDocumentWebSocket)
	r.GET("/ws/status", ws.HandleGlobalWebSocket)

	r.GET("/ws/notifications", func(c *gin.Context) {
		ws.HandleNotificationWS(c.Writer, c.Request)
	})

	// ⭐ MỚI THÊM — WS badge realtime
	r.GET("/ws/badge", func(c *gin.Context) {
		ws.HandleBadgeWS(c.Writer, c.Request)
	})

	// WebSocket goroutines
	go ws.HandleNotificationMessages()
	go ws.HandleBadgeMessages()
}
