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

	// ---------------- PAYMENT (MoMo VIP) ----------------
	payment := api.Group("/payment")
	{
		payment.GET("/packages", controllers.GetVIPPackages)
		payment.GET("/momo/callback", controllers.MoMoCallback)
		payment.POST("/momo/ipn", controllers.MoMoIPN)
		payment.GET("/status", controllers.CheckPaymentStatus)

		paymentProtected := payment.Group("/")
		paymentProtected.Use(middleware.AuthMiddleware())
		{
			paymentProtected.POST("/momo/create", controllers.CreateMoMoPayment)
			paymentProtected.GET("/history", controllers.GetPaymentHistory)
		}
	}

	// ---------------- USER ----------------
	user := api.Group("/users")
	user.Use(middleware.AuthMiddleware())
	{
		user.GET("/profile", controllers.GetProfile)
		user.PUT("/profile", controllers.UpdateProfile)
		user.POST("/change-password", controllers.ChangePassword)
		user.GET("/vip-status", controllers.GetUserVIPStatus)

		// ============ USER SAVED PODCASTS ============
		saved := user.Group("/saved")
		{
			saved.POST("/:id/toggle", controllers.ToggleLuuPodcast)
			saved.GET("", controllers.GetMySavedPodcasts)
		}

		// ============ USER FAVORITES ============
		favorites := user.Group("/favorites")
		{
			favorites.POST("/:id/toggle", controllers.ToggleYeuThichPodcast)
			favorites.GET("", controllers.GetMyFavoritePodcasts)
		}

		// ============ USER LISTENING HISTORY ============
		history := user.Group("/history")
		{
			history.POST("/save", controllers.LuuLichSuNghe)
			history.GET("", controllers.GetMyListeningHistory)
		}
	}

	// ============ USER NOTIFICATIONS ============
	userNotifications := api.Group("/user/notifications")
	userNotifications.Use(middleware.AuthMiddleware(), middleware.DBMiddleware(db))
	{
		userNotifications.GET("", controllers.GetNotifications)
		userNotifications.GET("/unread-count", controllers.GetUnreadCount)
		userNotifications.PUT("/:id/read", controllers.MarkNotificationAsRead)
		userNotifications.PUT("/read-all", controllers.MarkAllAsRead)
		userNotifications.DELETE("/:id", controllers.DeleteNotification)
		userNotifications.DELETE("", controllers.DeleteAllNotifications)
		userNotifications.DELETE("/read", controllers.DeleteReadNotifications)
	}

	// ---------------- ADMIN ----------------
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.DBMiddleware(db))
	{
		admin.GET("/vip-payments", controllers.GetAllVIPPayments(db))
		admin.GET("/vip-users", controllers.GetVIPUsers(db))
		admin.POST("/documents/upload", controllers.UploadDocument)
		admin.GET("/documents", controllers.ListDocumentStatus)
		admin.POST("/podcasts", controllers.CreatePodcastWithUpload)
		admin.PUT("/podcasts/:id", controllers.UpdatePodcast)
		admin.PATCH("/podcasts/:id/toggle-vip", controllers.TogglePodcastVIPStatus)
		admin.POST("/podcasts/sync-vip", controllers.SyncAllVIPStatus)
		admin.GET("/stats", controllers.GetAdminStats)
		admin.GET("/ratings/stats", controllers.GetAdminRatingsStats)
		admin.GET("/users", controllers.GetAllUsers)
		admin.PATCH("/users/:id/role", controllers.UpdateUserRole)
		admin.PATCH("/users/:id/toggle-active", controllers.ToggleUserActivation)

		adminNotif := admin.Group("/notifications")
		{
			adminNotif.GET("", controllers.GetAdminNotifications)
			adminNotif.GET("/unread-count", controllers.GetAdminUnreadCount)
			adminNotif.PUT("/:id/read", controllers.MarkAdminNotificationAsRead)
			adminNotif.PUT("/read-all", controllers.MarkAllAdminAsRead)
			adminNotif.DELETE("/:id", controllers.DeleteAdminNotification)
			adminNotif.DELETE("", controllers.DeleteAllAdminNotifications)
			adminNotif.DELETE("/read", controllers.DeleteReadAdminNotifications)
		}
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
	// ✅✅✅ CRITICAL: Specific routes MUST come BEFORE :id patterns
	publicPodcast := api.Group("/podcasts")
	{
		// Routes without :id parameter (safe to put anywhere)
		publicPodcast.GET("/", controllers.GetPodcast)
		publicPodcast.GET("/search", controllers.SearchPodcast)
		publicPodcast.GET("/vip", controllers.GetVIPPodcasts)
		publicPodcast.GET("/disabled", controllers.GetDisabledPodcasts)
		publicPodcast.GET("/featured", controllers.GetFeaturedPodcasts)

		// ✅✅✅ Specific :id routes MUST come BEFORE generic /:id
		// OptionalAuthMiddleware: parse token if present, but don't block
		publicPodcast.GET("/:id/check-vip", middleware.OptionalAuthMiddleware(), controllers.CheckPodcastVIPRequirement)
		publicPodcast.GET("/:id/ratings", controllers.GetPodcastRatings)
		publicPodcast.GET("/:id/recommendations", controllers.GetRecommendedPodcasts)

		// ✅ Generic :id route MUST be LAST
		publicPodcast.GET("/:id", middleware.OptionalAuthMiddleware(), controllers.GetPodcastByID)
	}

	// ---------------- AUTH REQUIRED PODCAST ----------------
	protectedPodcast := api.Group("/podcasts")
	protectedPodcast.Use(middleware.AuthMiddleware())
	{
		protectedPodcast.POST("/", controllers.CreatePodcastWithUpload)
		protectedPodcast.PUT("/:id", controllers.UpdatePodcast)
		protectedPodcast.POST("/:id/ratings", controllers.AddPodcastRating)
	}

	// ---------------- OTHER ----------------
	r.GET("/health", controllers.HealthCheck)

	// ---------------- WebSockets ----------------
	r.GET("/ws/document/:id", ws.HandleDocumentWebSocket)
	r.GET("/ws/status", ws.HandleGlobalWebSocket)
	r.GET("/ws/notifications", func(c *gin.Context) {
		ws.HandleNotificationWS(c.Writer, c.Request)
	})
	r.GET("/ws/badge", func(c *gin.Context) {
		ws.HandleBadgeWS(c.Writer, c.Request)
	})

	go ws.HandleNotificationMessages()
	go ws.HandleBadgeMessages()
}
