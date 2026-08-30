package handlers

import (
	"context"
	"strings"
	"sync"
	"time"
	"cboard/v2/internal/cache"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func roundToTwoDecimals(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

type userCustomNodeSummary struct {
	Count         int  `json:"custom_node_count"`
	DedicatedOnly bool `json:"dedicated_only"`
}

const (
	lineTypeNormal        = "normal"
	lineTypeDedicatedOnly = "dedicated_only"
	lineTypeMixed         = "mixed"
	lineTypeLegacyBoth    = "both"
)

func effectiveUserLineType(mode string, customNodeCount int, assignmentDedicatedOnly bool) string {
	switch strings.TrimSpace(mode) {
	case lineTypeNormal:
		return lineTypeNormal
	case lineTypeDedicatedOnly:
		return lineTypeDedicatedOnly
	case lineTypeMixed:
		if customNodeCount > 0 {
			return lineTypeMixed
		}
		return lineTypeNormal
	default:
		if customNodeCount == 0 {
			return lineTypeNormal
		}
		if assignmentDedicatedOnly {
			return lineTypeDedicatedOnly
		}
		return lineTypeMixed
	}
}

func loadUserCustomNodeSummaries(db *gorm.DB, userIDs []uint) map[uint]userCustomNodeSummary {
	summaries := make(map[uint]userCustomNodeSummary)
	if len(userIDs) == 0 {
		return summaries
	}

	var assignments []models.UserCustomNode
	db.Select("user_id, dedicated_only").Where("user_id IN ?", userIDs).Find(&assignments)
	for _, assignment := range assignments {
		summary := summaries[assignment.UserID]
		summary.Count++
		if assignment.DedicatedOnly {
			summary.DedicatedOnly = true
		}
		summaries[assignment.UserID] = summary
	}
	return summaries
}

// ==================== Dashboard ====================

func AdminDashboard(c *gin.Context) {
	var cachedData map[string]interface{}
	if cache.GetDashboardCache("admin_dashboard_stats", &cachedData) {
		utils.Success(c, cachedData)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	db := database.GetDB().WithContext(ctx)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	thirtyDaysAgo := todayStart.AddDate(0, 0, -29)

	var userCount, orderCount, subCount int64
	var pendingOrders, pendingTickets, newUsersToday int64
	var refundedOrders int64
	var revenueToday, revenueMonth float64
	var recentUsers []models.User
	var recentOrders []struct {
		models.Order
		UserEmail string `json:"user_email"`
	}
	var ticketList []models.Ticket

	type DayStat struct {
		Date  string  `json:"date"`
		Value float64 `json:"value"`
	}
	var revenueTrend []DayStat

	var wg sync.WaitGroup
	errCh := make(chan error, 12)
	runQuery := func(query func() error) {
		defer wg.Done()
		if err := query(); err != nil {
			errCh <- err
		}
	}
	wg.Add(13)

	go func() { runQuery(func() error { return db.Model(&models.User{}).Count(&userCount).Error }) }()
	go func() { runQuery(func() error { return db.Model(&models.Order{}).Count(&orderCount).Error }) }()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Subscription{}).Where("is_active = ? AND expire_time > ?", true, now).Count(&subCount).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Order{}).
				Where("status IN ? AND payment_time >= ? AND payment_time < ?", []string{models.OrderStatusPaid, models.OrderStatusCompleted}, todayStart, tomorrowStart).
				Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&revenueToday).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Order{}).
				Where("status IN ? AND payment_time >= ?", []string{models.OrderStatusPaid, models.OrderStatusCompleted}, monthStart).
				Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&revenueMonth).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Order{}).Where("status = ?", models.OrderStatusPending).Count(&pendingOrders).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Order{}).Where("status = ?", models.OrderStatusRefunded).Count(&refundedOrders).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Ticket{}).Where("status IN ?", []string{models.TicketStatusPending, models.TicketStatusOpen}).Count(&pendingTickets).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.User{}).Where("created_at >= ? AND created_at < ?", todayStart, tomorrowStart).Count(&newUsersToday).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Order("created_at DESC").Limit(5).Find(&recentUsers).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Table("orders").
				Select("orders.*, users.email as user_email").
				Joins("LEFT JOIN users ON users.id = orders.user_id").
				Order("orders.created_at DESC").
				Limit(5).
				Scan(&recentOrders).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Where("status IN ?", []string{models.TicketStatusPending, models.TicketStatusOpen}).Order("created_at DESC").Limit(5).Find(&ticketList).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Order{}).
				Where("status IN ? AND payment_time >= ?", []string{models.OrderStatusPaid, models.OrderStatusCompleted}, thirtyDaysAgo).
				Select("DATE(payment_time) as date, COALESCE(SUM(COALESCE(final_amount, amount)), 0) as value").
				Group("DATE(payment_time)").
				Order("date ASC").
				Scan(&revenueTrend).Error
		})
	}()

	wg.Wait()
	close(errCh)

	select {
	case <-ctx.Done():
		utils.InternalError(c, "查询超时")
		return
	default:
	}

	for err := range errCh {
		if err != nil {
			utils.InternalError(c, "获取仪表盘统计失败")
			return
		}
	}

	resultData := gin.H{
		"total_users":          userCount,
		"active_subscriptions": subCount,
		"today_revenue":        roundToTwoDecimals(revenueToday),
		"month_revenue":        roundToTwoDecimals(revenueMonth),
		"pending_orders":       pendingOrders,
		"refunded_count":       refundedOrders,
		"pending_tickets":      pendingTickets,
		"new_users_today":      newUsersToday,
		"recent_users":         recentUsers,
		"recent_orders":        recentOrders,
		"pending_ticket_list":  ticketList,
		"revenue_trend":        revenueTrend,
	}

	cache.SetDashboardCache("admin_dashboard_stats", resultData, 60*time.Second)
	utils.Success(c, resultData)
}

func AdminStats(c *gin.Context) {
	var cachedData map[string]interface{}
	if cache.GetDashboardCache("admin_stats_overview", &cachedData) {
		utils.Success(c, cachedData)
		return
	}

	db := database.GetDB()

	var userCount, activeUserCount, orderCount, paidOrderCount int64
	var subCount, activeSubCount, nodeCount, newUsersToday int64
	var totalRevenue float64

	today := time.Now().Format("2006-01-02")
	now := time.Now()

	var wg sync.WaitGroup
	errCh := make(chan error, 9)
	runQuery := func(query func() error) {
		defer wg.Done()
		if err := query(); err != nil {
			errCh <- err
		}
	}
	wg.Add(9)

	go func() { runQuery(func() error { return db.Model(&models.User{}).Count(&userCount).Error }) }()
	go func() {
		runQuery(func() error {
			return db.Model(&models.User{}).Where("is_active = ?", true).Count(&activeUserCount).Error
		})
	}()
	go func() { runQuery(func() error { return db.Model(&models.Order{}).Count(&orderCount).Error }) }()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Order{}).Where("status = ?", models.OrderStatusPaid).Count(&paidOrderCount).Error
		})
	}()
	go func() { runQuery(func() error { return db.Model(&models.Subscription{}).Count(&subCount).Error }) }()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Subscription{}).Where("is_active = ? AND expire_time > ?", true, now).Count(&activeSubCount).Error
		})
	}()
	go func() {
		runQuery(func() error { return db.Model(&models.Node{}).Where("is_active = ?", true).Count(&nodeCount).Error })
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Order{}).Where("status = ?", models.OrderStatusPaid).Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.User{}).Where("DATE(created_at) = ?", today).Count(&newUsersToday).Error
		})
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			utils.InternalError(c, "获取统计数据失败")
			return
		}
	}

	resultData := gin.H{
		"user_count":         userCount,
		"active_user_count":  activeUserCount,
		"new_users_today":    newUsersToday,
		"order_count":        orderCount,
		"paid_order_count":   paidOrderCount,
		"subscription_count": subCount,
		"active_sub_count":   activeSubCount,
		"node_count":         nodeCount,
		"total_revenue":      roundToTwoDecimals(totalRevenue),
	}

	cache.SetDashboardCache("admin_stats_overview", resultData, 60*time.Second)
	utils.Success(c, resultData)
}

// ==================== User Management ====================

func AdminMonitoring(c *gin.Context) {
	db := database.GetDB()
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	var nodeCount int64
	db.Model(&models.Node{}).Count(&nodeCount)
	var activeSubCount int64
	db.Model(&models.Subscription{}).Where("is_active = ? AND expire_time > ?", true, time.Now()).Count(&activeSubCount)
	var pendingTickets int64
	db.Model(&models.Ticket{}).Where("status = ?", models.TicketStatusPending).Count(&pendingTickets)
	var pendingOrders int64
	db.Model(&models.Order{}).Where("status = ?", models.OrderStatusPending).Count(&pendingOrders)
	utils.Success(c, gin.H{
		"user_count":           userCount,
		"node_count":           nodeCount,
		"active_subscriptions": activeSubCount,
		"pending_tickets":      pendingTickets,
		"pending_orders":       pendingOrders,
	})
}

