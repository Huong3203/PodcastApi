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

	// ---------------- MOMO VIP ----------------
	// ---------------- MOMO VIP ----------------
	// momo := api.Group("/momo")
	// {
	// 	// Public routes - FE và MoMo gọi
	// 	momo.POST("/vip/create", controllers.CreateMomoVIPPayment(db))
	// 	momo.POST("/vip/ipn", controllers.MomoIPN(db))       // ĐÃ SỬA: MomoVIPIPN → MomoIPN
	// 	momo.GET("/vip/return", controllers.MomoReturnURL()) // ĐÃ SỬA: MomoVIPReturn → MomoReturnURL

	// 	// Check payment status
	// 	//momo.GET("/payment/status/:orderId", controllers.CheckPaymentStatus(db))
	// 	//momo.POST("/payment/verify/:orderId", controllers.VerifyPaymentAndSetVIP(db))

	// 	// Force complete (debug)
	// 	//momo.POST("/payment/force-complete/:orderId", controllers.ForceCompletePayment(db))

	// 	// Protected routes
	// 	momoProtected := momo.Group("/vip")
	// 	momoProtected.Use(middleware.AuthMiddleware())
	// 	{
	// 		momoProtected.GET("/history/:userId", controllers.GetUserVIPHistory(db))
	// 		momoProtected.GET("/status/:userId", controllers.CheckUserVIP(db))
	// 	}
	// }

	// ---------------- PAYMENT (MoMo VIP) ----------------
	payment := api.Group("/payment")
	{
		// Public routes - không cần auth
		payment.GET("/packages", controllers.GetVIPPackages)    // Danh sách gói VIP
		payment.GET("/momo/callback", controllers.MoMoCallback) // Redirect từ MoMo
		payment.POST("/momo/ipn", controllers.MoMoIPN)          // Webhook từ MoMo
		payment.GET("/status", controllers.CheckPaymentStatus)

		// Protected routes - cần đăng nhập
		paymentProtected := payment.Group("/")
		paymentProtected.Use(middleware.AuthMiddleware())
		{
			paymentProtected.POST("/momo/create", controllers.CreateMoMoPayment) // Tạo thanh toán
			paymentProtected.GET("/history", controllers.GetPaymentHistory)      // Lịch sử thanh toán
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

	r.GET("/ws/badge", func(c *gin.Context) {
		ws.HandleBadgeWS(c.Writer, c.Request)
	})

	// WebSocket goroutines
	go ws.HandleNotificationMessages()
	go ws.HandleBadgeMessages()
}
