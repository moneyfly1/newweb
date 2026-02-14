package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cboard/v2/internal/api/handlers"
	"cboard/v2/internal/api/middleware"
	"cboard/v2/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())

	// CORS
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	if len(cfg.CorsOrigins) > 0 {
		corsConfig.AllowOrigins = cfg.CorsOrigins
	} else {
		corsConfig.AllowOrigins = []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:3080", "http://localhost:8000", "http://localhost:9000"}
	}
	r.Use(cors.New(corsConfig))

	api := r.Group("/api/v1")

	// ===== 公开路由（无需认证）=====
	auth := api.Group("/auth")
	{
		auth.POST("/register", middleware.RateLimit(5, time.Minute), handlers.Register)
		auth.POST("/login", middleware.RateLimit(10, time.Minute), handlers.Login)
		auth.POST("/refresh", handlers.RefreshToken)
		auth.POST("/logout", middleware.AuthRequired(), handlers.Logout)
		auth.POST("/verification/send", middleware.RateLimit(3, time.Minute), handlers.SendVerificationCode)
		auth.POST("/verification/verify", handlers.VerifyCode)
		auth.POST("/forgot-password", middleware.RateLimit(3, time.Minute), handlers.ForgotPassword)
		auth.POST("/reset-password", handlers.ResetPassword)
	}

	// 公开订阅链接
	api.GET("/sub/clash/:url", handlers.GetSubscription)
	api.GET("/sub/:url", handlers.GetUniversalSubscription)
	// 兼容旧路径
	api.GET("/subscribe/clash/:url", handlers.GetSubscription)
	api.GET("/subscribe/universal/:url", handlers.GetUniversalSubscription)
	api.GET("/subscribe/:url", handlers.GetSubscription)

	// 公开配置
	api.GET("/config", handlers.GetPublicConfig)
	api.GET("/packages", handlers.ListPackages)
	api.GET("/packages/:id", handlers.GetPackage)
	api.GET("/announcements", handlers.ListPublicAnnouncements)

	// 支付回调（无需认证）
	api.POST("/payment/notify/:type", handlers.PaymentNotify)
	api.GET("/payment/notify/:type", handlers.PaymentNotify)
	api.GET("/payment/methods", handlers.GetPaymentMethods)

	// 邀请码验证（公开）
	api.GET("/invites/validate/:code", handlers.ValidateInviteCode)

	// ===== 需要认证的路由 =====
	authorized := api.Group("")
	authorized.Use(middleware.AuthRequired())
	{
		// 用户
		users := authorized.Group("/users")
		{
			users.GET("/me", handlers.GetCurrentUser)
			users.PUT("/me", handlers.UpdateCurrentUser)
			users.POST("/change-password", handlers.ChangePassword)
			users.PUT("/preferences", handlers.UpdatePreferences)
			users.GET("/notification-settings", handlers.GetNotificationSettings)
			users.PUT("/notification-settings", handlers.UpdateNotificationSettings)
			users.GET("/privacy-settings", handlers.GetPrivacySettings)
			users.PUT("/privacy-settings", handlers.UpdatePrivacySettings)
			users.GET("/my-level", handlers.GetMyLevel)
			users.GET("/login-history", handlers.GetLoginHistory)
			users.GET("/activities", handlers.GetActivities)
			users.GET("/dashboard-info", handlers.GetDashboardInfo)
			users.GET("/subscription-resets", handlers.GetSubscriptionResets)
			users.GET("/devices", handlers.GetUserDevices)
		}

		// 订阅
		subs := authorized.Group("/subscriptions")
		{
			subs.GET("/user-subscription", handlers.GetUserSubscription)
			subs.GET("/devices", handlers.GetSubscriptionDevices)
			subs.POST("/reset-subscription", handlers.ResetSubscription)
			subs.POST("/convert-to-balance", handlers.ConvertToBalance)
			subs.POST("/send-subscription-email", handlers.SendSubscriptionEmail)
			subs.DELETE("/devices/:id", handlers.DeleteSubscriptionDevice)
		}

		// 订单
		orders := authorized.Group("/orders")
		{
			orders.GET("", handlers.ListOrders)
			orders.POST("", handlers.CreateOrder)
			orders.POST("/:orderNo/pay", handlers.PayOrder)
			orders.POST("/:orderNo/cancel", handlers.CancelOrder)
			orders.GET("/:orderNo/status", handlers.GetOrderStatus)
		}

		// 支付
		authorized.POST("/payment", handlers.CreatePayment)
		authorized.GET("/payment/status/:id", handlers.GetPaymentStatus)

		// 卡密兑换
		authorized.POST("/redeem", handlers.RedeemCode)
		authorized.GET("/redeem/history", handlers.GetRedeemHistory)

		// 节点
		nodes := authorized.Group("/nodes")
		{
			nodes.GET("", handlers.ListNodes)
			nodes.GET("/stats", handlers.GetNodeStats)
			nodes.GET("/:id", handlers.GetNode)
			nodes.POST("/:id/test", handlers.TestNode)
			nodes.POST("/batch-test", handlers.BatchTestNodes)
		}

		// 优惠券
		authorized.POST("/coupons/verify", handlers.VerifyCoupon)
		authorized.GET("/coupons/my", handlers.GetMyCoupons)

		// 通知
		notifs := authorized.Group("/notifications")
		{
			notifs.GET("", handlers.ListNotifications)
			notifs.GET("/unread-count", handlers.GetUnreadCount)
			notifs.PUT("/:id/read", handlers.MarkNotificationRead)
			notifs.PUT("/read-all", handlers.MarkAllNotificationsRead)
			notifs.DELETE("/:id", handlers.DeleteNotification)
		}

		// 工单
		tickets := authorized.Group("/tickets")
		{
			tickets.GET("", handlers.ListTickets)
			tickets.POST("", handlers.CreateTicket)
			tickets.GET("/:id", handlers.GetTicket)
			tickets.POST("/:id/reply", handlers.ReplyTicket)
			tickets.PUT("/:id", handlers.CloseTicket)
		}

		// 设备
		authorized.GET("/devices", handlers.ListDevices)
		authorized.DELETE("/devices/:id", handlers.DeleteDevice)

		// 邀请
		invites := authorized.Group("/invites")
		{
			invites.GET("", handlers.ListInviteCodes)
			invites.POST("", handlers.CreateInviteCode)
			invites.GET("/stats", handlers.GetInviteStats)
			invites.GET("/my-codes", handlers.GetMyCodes)
			invites.DELETE("/:id", handlers.DeleteInviteCode)
		}

		// 充值
		recharge := authorized.Group("/recharge")
		{
			recharge.GET("", handlers.ListRechargeRecords)
			recharge.POST("", handlers.CreateRecharge)
			recharge.POST("/:id/pay", handlers.CreateRechargePayment)
			recharge.POST("/:id/cancel", handlers.CancelRecharge)
		}
	}

	// ===== 管理员路由 =====
	admin := api.Group("/admin")
	admin.Use(middleware.AuthRequired(), middleware.AdminRequired())
	{
		// 仪表盘
		admin.GET("/dashboard", handlers.AdminDashboard)
		admin.GET("/stats", handlers.AdminStats)

		// 用户管理
		adminUsers := admin.Group("/users")
		{
			adminUsers.GET("", handlers.AdminListUsers)
			adminUsers.POST("", handlers.AdminCreateUser)
			adminUsers.GET("/:id", handlers.AdminGetUser)
			adminUsers.PUT("/:id", handlers.AdminUpdateUser)
			adminUsers.DELETE("/:id", handlers.AdminDeleteUser)
			adminUsers.POST("/:id/toggle-active", handlers.AdminToggleUserActive)
			adminUsers.POST("/:id/reset-password", handlers.AdminResetUserPassword)
			adminUsers.GET("/abnormal", handlers.AdminGetAbnormalUsers)
			adminUsers.POST("/:id/login-as", handlers.AdminLoginAsUser)
			adminUsers.POST("/batch-action", handlers.AdminBatchUserAction)
		}

		// 订单管理
		adminOrders := admin.Group("/orders")
		{
			adminOrders.GET("", handlers.AdminListOrders)
			adminOrders.GET("/:id", handlers.AdminGetOrder)
			adminOrders.POST("/:id/refund", handlers.AdminRefundOrder)
		}

		// 套餐管理
		adminPkgs := admin.Group("/packages")
		{
			adminPkgs.GET("", handlers.AdminListPackages)
			adminPkgs.POST("", handlers.AdminCreatePackage)
			adminPkgs.PUT("/:id", handlers.AdminUpdatePackage)
			adminPkgs.DELETE("/:id", handlers.AdminDeletePackage)
		}

		// 节点管理
		adminNodes := admin.Group("/nodes")
		{
			adminNodes.GET("", handlers.AdminListNodes)
			adminNodes.POST("", handlers.AdminCreateNode)
			adminNodes.PUT("/:id", handlers.AdminUpdateNode)
			adminNodes.DELETE("/:id", handlers.AdminDeleteNode)
			adminNodes.POST("/import", handlers.AdminImportNodes)
			adminNodes.POST("/:id/test", handlers.AdminTestNode)
			adminNodes.POST("/batch-action", handlers.AdminBatchNodeAction)
		}

		// 专线节点
		adminCustomNodes := admin.Group("/custom-nodes")
		{
			adminCustomNodes.GET("", handlers.AdminListCustomNodes)
			adminCustomNodes.POST("", handlers.AdminCreateCustomNode)
			adminCustomNodes.PUT("/:id", handlers.AdminUpdateCustomNode)
			adminCustomNodes.DELETE("/:id", handlers.AdminDeleteCustomNode)
			adminCustomNodes.POST("/:id/assign", handlers.AdminAssignCustomNode)
			adminCustomNodes.POST("/import-links", handlers.AdminImportCustomNodeLinks)
			adminCustomNodes.POST("/batch-delete", handlers.AdminBatchDeleteCustomNodes)
			adminCustomNodes.GET("/:id/link", handlers.AdminGetCustomNodeLink)
			adminCustomNodes.GET("/:id/users", handlers.AdminGetCustomNodeUsers)
		}

		// 节点自动更新
		configUpdate := admin.Group("/config-update")
		{
			configUpdate.GET("/status", handlers.AdminConfigUpdateStatus)
			configUpdate.GET("/config", handlers.AdminGetConfigUpdateConfig)
			configUpdate.PUT("/config", handlers.AdminSaveConfigUpdateConfig)
			configUpdate.POST("/start", handlers.AdminStartConfigUpdate)
			configUpdate.POST("/stop", handlers.AdminStopConfigUpdate)
			configUpdate.GET("/logs", handlers.AdminGetConfigUpdateLogs)
			configUpdate.POST("/logs/clear", handlers.AdminClearConfigUpdateLogs)
		}

		// 订阅管理
		adminSubs := admin.Group("/subscriptions")
		{
			adminSubs.GET("", handlers.AdminListSubscriptions)
			adminSubs.GET("/:id", handlers.AdminGetSubscription)
			adminSubs.POST("/:id/reset", handlers.AdminResetSubscription)
			adminSubs.POST("/:id/extend", handlers.AdminExtendSubscription)
			adminSubs.PUT("/:id", handlers.AdminUpdateSubscription)
			adminSubs.POST("/:id/send-email", handlers.AdminSendSubscriptionEmail)
			adminSubs.POST("/:id/set-expire", handlers.AdminSetSubscriptionExpireTime)
		}

		// 用户完全删除
		admin.DELETE("/users/:id/full", handlers.AdminDeleteUserFull)

		// 优惠券管理
		adminCoupons := admin.Group("/coupons")
		{
			adminCoupons.GET("", handlers.AdminListCoupons)
			adminCoupons.POST("", handlers.AdminCreateCoupon)
			adminCoupons.PUT("/:id", handlers.AdminUpdateCoupon)
			adminCoupons.DELETE("/:id", handlers.AdminDeleteCoupon)
		}

		// 工单管理
		adminTickets := admin.Group("/tickets")
		{
			adminTickets.GET("", handlers.AdminListTickets)
			adminTickets.GET("/:id", handlers.AdminGetTicket)
			adminTickets.PUT("/:id", handlers.AdminUpdateTicket)
			adminTickets.POST("/:id/reply", handlers.AdminReplyTicket)
		}

		// 用户等级
		adminLevels := admin.Group("/user-levels")
		{
			adminLevels.GET("", handlers.AdminListUserLevels)
			adminLevels.POST("", handlers.AdminCreateUserLevel)
			adminLevels.PUT("/:id", handlers.AdminUpdateUserLevel)
			adminLevels.DELETE("/:id", handlers.AdminDeleteUserLevel)
		}

		// 卡密管理
		adminRedeem := admin.Group("/redeem-codes")
		{
			adminRedeem.GET("", handlers.AdminListRedeemCodes)
			adminRedeem.POST("", handlers.AdminCreateRedeemCodes)
			adminRedeem.DELETE("/:id", handlers.AdminDeleteRedeemCode)
		}

		// 邮件队列
		admin.GET("/email-queue", handlers.AdminListEmailQueue)
		admin.POST("/email-queue/:id/retry", handlers.AdminRetryEmail)
		admin.DELETE("/email-queue/:id", handlers.AdminDeleteEmail)

		// 系统设置
		settings := admin.Group("/settings")
		{
			settings.GET("", handlers.AdminGetSettings)
			settings.PUT("", handlers.AdminUpdateSettings)
			settings.POST("/test-email", handlers.AdminSendTestEmail)
		}

		// 公告
		announcements := admin.Group("/announcements")
		{
			announcements.GET("", handlers.AdminListAnnouncements)
			announcements.POST("", handlers.AdminCreateAnnouncement)
			announcements.PUT("/:id", handlers.AdminUpdateAnnouncement)
			announcements.DELETE("/:id", handlers.AdminDeleteAnnouncement)
		}

		// 统计
		admin.GET("/stats/revenue", handlers.AdminRevenueStats)
		admin.GET("/stats/users", handlers.AdminUserStats)
		admin.GET("/stats/regions", handlers.AdminRegionStats)

		// 日志
		admin.GET("/logs/audit", handlers.AdminAuditLogs)
		admin.GET("/logs/login", handlers.AdminLoginLogs)
		admin.GET("/logs/registration", handlers.AdminRegistrationLogs)
		admin.GET("/logs/subscription", handlers.AdminSubscriptionLogs)
		admin.GET("/logs/balance", handlers.AdminBalanceLogs)
		admin.GET("/logs/commission", handlers.AdminCommissionLogs)

		// 监控
		admin.GET("/monitoring", handlers.AdminMonitoring)

		// 备份
		admin.POST("/backup", handlers.AdminCreateBackup)
		admin.GET("/backup", handlers.AdminListBackups)
	}

	// Serve frontend static files and SPA fallback
	distPath := filepath.Join("frontend", "dist")
	if _, err := os.Stat(distPath); err == nil {
		r.Static("/assets", filepath.Join(distPath, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(distPath, "favicon.ico"))

		// SPA fallback: any route not matched by API or static files serves index.html
		indexHTML := filepath.Join(distPath, "index.html")
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// Don't serve index.html for API routes
			if strings.HasPrefix(path, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "not found"})
				return
			}
			c.File(indexHTML)
		})
	}

	return r
}