// routes/routes.go
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
	{
		user.Use(middleware.AuthMiddleware())
		user.GET("/profile", controllers.GetProfile)
		user.PUT("/profile", controllers.UpdateProfile)
		user.POST("/change-password", controllers.ChangePassword)
	}

	// ============ USER LISTENING HISTORY ============
	userHistory := api.Group("/user/listening-history")
	userHistory.Use(middleware.AuthMiddleware(), middleware.DBMiddleware(db))
	{
		userHistory.POST("/:podcast_id", controllers.SavePodcastHistory)     // Lưu lịch sử
		userHistory.GET("", controllers.GetListeningHistory)                 // Lấy danh sách
		userHistory.GET("/:podcast_id", controllers.GetPodcastHistory)       // Lấy 1 podcast
		userHistory.DELETE("/:podcast_id", controllers.DeletePodcastHistory) // Xóa 1 podcast
		userHistory.DELETE("", controllers.ClearAllHistory)                  // Xóa tất cả
	}

	// ============ USER FAVORITES ============
	// userFavorites := api.Group("/user/favorites")
	// userFavorites.Use(middleware.AuthMiddleware(), middleware.DBMiddleware(db))
	// {
	// 	userFavorites.POST("/:podcast_id", controllers.AddFavorite)        // Thêm yêu thích
	// 	userFavorites.DELETE("/:podcast_id", controllers.RemoveFavorite)   // Bỏ yêu thích
	// 	userFavorites.GET("/:podcast_id/check", controllers.CheckFavorite) // Kiểm tra
	// 	userFavorites.GET("", controllers.GetFavorites)                    // Danh sách
	// }

	// ============ USER NOTIFICATIONS ============
	userNotifications := api.Group("/user/notifications")
	userNotifications.Use(middleware.AuthMiddleware(), middleware.DBMiddleware(db))
	{
		userNotifications.GET("", controllers.GetNotifications)                // Danh sách
		userNotifications.GET("/unread-count", controllers.GetUnreadCount)     // Đếm chưa đọc
		userNotifications.PUT("/:id/read", controllers.MarkNotificationAsRead) // Đánh dấu đã đọc
		userNotifications.PUT("/read-all", controllers.MarkAllAsRead)          // Đánh dấu tất cả
		userNotifications.DELETE("/:id", controllers.DeleteNotification)       // Xóa 1 cái
		userNotifications.DELETE("", controllers.DeleteAllNotifications)       // Xóa tất cả
		userNotifications.DELETE("/read", controllers.DeleteReadNotifications) // Xóa đã đọc
	}

	// ---------------- ADMIN ----------------
	admin := api.Group("/admin")
	{
		admin.Use(middleware.AuthMiddleware(), middleware.DBMiddleware(db))

		// Admin VIP management
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

		// ============ ADMIN NOTIFICATIONS ============
		adminNotif := admin.Group("/notifications")
		{
			adminNotif.GET("", controllers.GetAdminNotifications)                // Danh sách
			adminNotif.GET("/unread-count", controllers.GetAdminUnreadCount)     // Đếm chưa đọc
			adminNotif.PUT("/:id/read", controllers.MarkAdminNotificationAsRead) // Đánh dấu đã đọc
			adminNotif.PUT("/read-all", controllers.MarkAllAdminAsRead)          // Đánh dấu tất cả
			adminNotif.DELETE("/:id", controllers.DeleteAdminNotification)       // Xóa 1 cái
			adminNotif.DELETE("", controllers.DeleteAllAdminNotifications)       // Xóa tất cả
			adminNotif.DELETE("/read", controllers.DeleteReadAdminNotifications) // Xóa đã đọc
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
