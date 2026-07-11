package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"skintrader-go/internal/config"
	"skintrader-go/internal/handler"
	"skintrader-go/internal/middleware"
)

type Handlers struct {
	Auth         *handler.AuthHandler
	User         *handler.UserHandler
	Post         *handler.PostHandler
	Game         *handler.GameHandler
	Message      *handler.MessageHandler
	Subscription *handler.SubscriptionHandler
	Payment      *handler.PaymentHandler
	Report       *handler.ReportHandler
	Admin        *handler.AdminHandler
}

type Dependencies struct {
	AuthMiddleware *middleware.AuthMiddleware
	RateLimiter    *middleware.RateLimiter
}

func Setup(r *gin.Engine, cfg *config.Config, logger zerolog.Logger, h *Handlers, deps *Dependencies) {
	// Global middleware
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logger(logger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.CORS.Origins))
	r.Use(middleware.Language())
	r.Use(middleware.RequestSizeLimit(15 * 1024 * 1024)) // 15MB (10MB file cap + multipart overhead headroom)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Serve uploaded files (except KYC)
	r.Static("/uploads/posts", cfg.Upload.Dir+"/posts")
	r.Static("/uploads/avatars", cfg.Upload.Dir+"/avatars")

	// Block direct KYC access
	r.GET("/uploads/kyc/*filepath", func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Direct access to KYC documents is not allowed",
		})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		setupAuthRoutes(v1, h, deps)
		setupUserRoutes(v1, h, deps)
		setupPostRoutes(v1, h, deps)
		setupGameRoutes(v1, h, deps)
		setupMessageRoutes(v1, h, deps)
		setupSubscriptionRoutes(v1, h, deps)
		setupPaymentRoutes(v1, h, deps)
		setupReportRoutes(v1, h, deps)
		setupAdminRoutes(v1, h, deps)
	}

	// 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Route not found",
			"code":    "NOT_FOUND",
		})
	})
}

func setupAuthRoutes(rg *gin.RouterGroup, h *Handlers, deps *Dependencies) {
	auth := rg.Group("/auth")
	{
		// Public
		auth.POST("/google", deps.RateLimiter.Limit(middleware.StrictLimit), h.Auth.GoogleAuth)
		auth.POST("/apple", deps.RateLimiter.Limit(middleware.StrictLimit), h.Auth.AppleAuth)
		auth.POST("/register", deps.RateLimiter.Limit(middleware.StrictLimit), h.Auth.Register)
		auth.POST("/login", deps.RateLimiter.Limit(middleware.StrictLimit), h.Auth.Login)
		auth.POST("/refresh-token", deps.RateLimiter.Limit(middleware.StrictLimit), h.Auth.RefreshToken)

		// Authenticated
		authenticated := auth.Group("", deps.AuthMiddleware.AuthenticateUser())
		{
			authenticated.POST("/logout", h.Auth.Logout)
			authenticated.POST("/logout-all", h.Auth.LogoutAll)
			authenticated.GET("/me", h.Auth.GetMe)

			// KYC
			authenticated.POST("/kyc/upload", deps.RateLimiter.Limit(middleware.UploadLimit), h.Auth.UploadKYCDocument)
			authenticated.POST("/kyc/verify", h.Auth.VerifyKYC)
			authenticated.GET("/kyc/status", h.Auth.GetKYCStatus)
		}
	}
}

func setupUserRoutes(rg *gin.RouterGroup, h *Handlers, deps *Dependencies) {
	users := rg.Group("/users")
	{
		// Public
		users.GET("/:id", h.User.GetPublicProfile)
		users.GET("/:id/posts", h.User.GetUserPosts)

		// Authenticated
		authenticated := users.Group("", deps.AuthMiddleware.AuthenticateUser())
		{
			authenticated.GET("/profile", h.User.GetProfile)
			authenticated.PUT("/profile", h.User.UpdateProfile)
			authenticated.PUT("/profile/avatar", deps.RateLimiter.Limit(middleware.UploadLimit), h.User.UpdateAvatar)
			authenticated.PUT("/location", h.User.UpdateLocation)
			authenticated.GET("/nearby", h.User.GetNearbyUsers)
			authenticated.DELETE("/account", h.User.DeleteAccount)
		}
	}
}

func setupPostRoutes(rg *gin.RouterGroup, h *Handlers, deps *Dependencies) {
	posts := rg.Group("/posts")
	{
		// Public (with optional auth)
		// Registered without trailing slash: Gin's trailing-slash redirect bypasses
		// middleware, so the redirected response would lack CORS headers.
		posts.GET("", deps.AuthMiddleware.OptionalAuth(), h.Post.GetPosts)
		posts.GET("/search", deps.RateLimiter.Limit(middleware.SearchLimit), deps.AuthMiddleware.OptionalAuth(), h.Post.SearchPosts)
		posts.GET("/:id", deps.AuthMiddleware.OptionalAuth(), h.Post.GetPost)

		// Authenticated
		authenticated := posts.Group("", deps.AuthMiddleware.AuthenticateUser())
		{
			authenticated.GET("/my", h.Post.GetMyPosts)
			authenticated.POST("", deps.RateLimiter.Limit(middleware.UploadLimit), h.Post.CreatePost)
			authenticated.PUT("/:id", deps.RateLimiter.Limit(middleware.UploadLimit), h.Post.UpdatePost)
			authenticated.PATCH("/:id/status", h.Post.UpdatePostStatus)
			authenticated.DELETE("/:id", h.Post.DeletePost)
			authenticated.POST("/:id/images", deps.RateLimiter.Limit(middleware.UploadLimit), h.Post.AddImages)
			authenticated.DELETE("/:id/images/:imageId", h.Post.RemoveImage)
		}
	}
}

func setupGameRoutes(rg *gin.RouterGroup, h *Handlers, deps *Dependencies) {
	games := rg.Group("/games")
	{
		games.GET("", h.Game.GetGames)
		games.GET("/search", deps.RateLimiter.Limit(middleware.SearchLimit), h.Game.SearchGames)
		games.GET("/popular", h.Game.GetPopularGames)
		games.GET("/genres", h.Game.GetGenres)
		games.GET("/genre/:genre", h.Game.GetGamesByGenre)
		games.GET("/:identifier", h.Game.GetGame)
	}
}

func setupMessageRoutes(rg *gin.RouterGroup, h *Handlers, deps *Dependencies) {
	messages := rg.Group("/messages", deps.AuthMiddleware.AuthenticateUser())
	{
		messages.POST("/send", deps.RateLimiter.Limit(middleware.MessageLimit), h.Message.SendMessage)
		messages.GET("/conversations", h.Message.GetConversations)
		messages.POST("/conversations/start", h.Message.StartConversation)
		messages.GET("/conversations/:id", h.Message.GetConversationMessages)
		messages.DELETE("/conversations/:id", h.Message.DeleteConversation)
		messages.PATCH("/read/:conversationId", h.Message.MarkAsRead)
		messages.DELETE("/:id", h.Message.DeleteMessage)
		messages.GET("/unread-count", h.Message.GetUnreadCount)
	}
}

func setupSubscriptionRoutes(rg *gin.RouterGroup, h *Handlers, deps *Dependencies) {
	subs := rg.Group("/subscriptions", deps.AuthMiddleware.AuthenticateUser())
	{
		subs.GET("/status", h.Subscription.GetStatus)
		subs.POST("/initiate", h.Subscription.Initiate)
		subs.GET("/history", h.Subscription.GetHistory)
		subs.POST("/cancel", h.Subscription.Cancel)
	}
}

func setupPaymentRoutes(rg *gin.RouterGroup, h *Handlers, deps *Dependencies) {
	payments := rg.Group("/payments")
	{
		// Payment gateway webhooks (public, verified by gateway-specific auth)
		payments.POST("/payme/webhook", h.Payment.HandlePaymeWebhook)
		payments.GET("/payme/callback", h.Payment.HandlePaymeCallback)
		payments.POST("/click/prepare", h.Payment.HandleClickPrepare)
		payments.POST("/click/complete", h.Payment.HandleClickComplete)
		payments.POST("/xazna/webhook", h.Payment.HandleXaznaWebhook)
		payments.POST("/uzum/webhook", h.Payment.HandleUzumWebhook)

		// User transactions (authenticated)
		authenticated := payments.Group("", deps.AuthMiddleware.AuthenticateUser())
		{
			authenticated.GET("/transactions", h.Payment.GetTransactions)
			authenticated.GET("/transactions/:id", h.Payment.GetTransactionByID)
		}
	}
}

func setupReportRoutes(rg *gin.RouterGroup, h *Handlers, deps *Dependencies) {
	reports := rg.Group("/reports", deps.AuthMiddleware.AuthenticateUser())
	{
		reports.POST("", h.Report.CreateReport)
		reports.GET("/my", h.Report.GetMyReports)
		reports.GET("/:id", h.Report.GetReportByID)
	}
}

func setupAdminRoutes(rg *gin.RouterGroup, h *Handlers, deps *Dependencies) {
	admin := rg.Group("/admin")
	{
		// Public admin auth
		admin.POST("/login", deps.RateLimiter.Limit(middleware.StrictLimit), h.Admin.Login)
		admin.POST("/refresh-token", deps.RateLimiter.Limit(middleware.StrictLimit), h.Admin.RefreshToken)

		// Authenticated admin routes
		authenticated := admin.Group("", deps.AuthMiddleware.AuthenticateAdmin(), deps.RateLimiter.Limit(middleware.AdminLimit))
		{
			authenticated.POST("/logout", h.Admin.Logout)

			// Dashboard
			authenticated.GET("/stats", h.Admin.GetStats)

			// Users
			authenticated.GET("/users", h.Admin.GetUsers)
			authenticated.GET("/users/:id", h.Admin.GetUserDetails)
			authenticated.PATCH("/users/:id/status", h.Admin.UpdateUserStatus)

			// Posts
			authenticated.GET("/posts", h.Admin.GetPosts)
			authenticated.DELETE("/posts/:id", h.Admin.DeletePost)

			// KYC
			authenticated.GET("/kyc/pending", h.Admin.GetPendingKYC)
			authenticated.PATCH("/kyc/:userId/approve", h.Admin.ApproveKYC)
			authenticated.PATCH("/kyc/:userId/reject", h.Admin.RejectKYC)
			authenticated.GET("/kyc/image/:filename", h.Admin.ServeKYCImage)

			// Logs
			authenticated.GET("/logs", h.Admin.GetLogs)

			// Admin management
			authenticated.GET("/admins", h.Admin.GetAdmins)
			authenticated.POST("/admins", h.Admin.CreateAdmin)
			authenticated.PUT("/admins/:id", h.Admin.UpdateAdmin)

			// Games
			authenticated.GET("/games", h.Admin.GetGames)
			authenticated.POST("/games", h.Admin.CreateGame)
			authenticated.PUT("/games/:id", h.Admin.UpdateGame)
			authenticated.DELETE("/games/:id", h.Admin.DeleteGame)

			// Subscriptions
			authenticated.GET("/subscriptions", h.Admin.GetSubscriptions)
			authenticated.GET("/subscriptions/stats", h.Admin.GetSubscriptionStats)
			authenticated.POST("/subscriptions/grant", h.Admin.GrantSubscription)
			authenticated.POST("/subscriptions/:id/revoke", h.Admin.RevokeSubscription)
			authenticated.GET("/transactions", h.Admin.GetTransactions)

			// Reports
			authenticated.GET("/reports", h.Admin.GetReports)
			authenticated.GET("/reports/stats", h.Admin.GetReportStats)
			authenticated.GET("/reports/:id", h.Admin.GetReportDetails)
			authenticated.PATCH("/reports/:id/status", h.Admin.UpdateReportStatus)
			authenticated.POST("/reports/:id/resolve", h.Admin.ResolveReport)
			authenticated.POST("/reports/:id/dismiss", h.Admin.DismissReport)
		}
	}
}
