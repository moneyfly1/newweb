package handlers

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/services/git"
	"cboard/v2/internal/utils"
	"cboard/v2/internal/cache"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== Dashboard ====================

func AdminDashboard(c *gin.Context) {
	var cachedData map[string]interface{}
	if cache.GetDashboardCache("admin_dashboard_stats", &cachedData) {
		utils.Success(c, cachedData)
		return
	}

	db := database.GetDB()

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	thirtyDaysAgo := todayStart.AddDate(0, 0, -29)

	var userCount, orderCount, subCount int64
	var pendingOrders, pendingTickets, newUsersToday int64
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
	wg.Add(12)

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
				Where("status IN ? AND payment_time >= ? AND payment_time < ?", []string{"paid", "completed"}, todayStart, tomorrowStart).
				Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&revenueToday).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Order{}).
				Where("status IN ? AND payment_time >= ?", []string{"paid", "completed"}, monthStart).
				Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&revenueMonth).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Order{}).Where("status = ?", "pending").Count(&pendingOrders).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Ticket{}).Where("status IN ?", []string{"pending", "open"}).Count(&pendingTickets).Error
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
			return db.Where("status IN ?", []string{"pending", "open"}).Order("created_at DESC").Limit(5).Find(&ticketList).Error
		})
	}()
	go func() {
		runQuery(func() error {
			return db.Model(&models.Order{}).
				Where("status IN ? AND payment_time >= ?", []string{"paid", "completed"}, thirtyDaysAgo).
				Select("DATE(payment_time) as date, COALESCE(SUM(COALESCE(final_amount, amount)), 0) as value").
				Group("DATE(payment_time)").
				Order("date ASC").
				Scan(&revenueTrend).Error
		})
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			utils.InternalError(c, "获取仪表盘统计失败")
			return
		}
	}

	resultData := gin.H{
		"total_users":          userCount,
		"active_subscriptions": subCount,
		"today_revenue":        revenueToday,
		"month_revenue":        revenueMonth,
		"pending_orders":       pendingOrders,
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
	go func() { runQuery(func() error { return db.Model(&models.User{}).Where("is_active = ?", true).Count(&activeUserCount).Error }) }()
	go func() { runQuery(func() error { return db.Model(&models.Order{}).Count(&orderCount).Error }) }()
	go func() { runQuery(func() error { return db.Model(&models.Order{}).Where("status = ?", "paid").Count(&paidOrderCount).Error }) }()
	go func() { runQuery(func() error { return db.Model(&models.Subscription{}).Count(&subCount).Error }) }()
	go func() { runQuery(func() error { return db.Model(&models.Subscription{}).Where("is_active = ? AND expire_time > ?", true, now).Count(&activeSubCount).Error }) }()
	go func() { runQuery(func() error { return db.Model(&models.Node{}).Where("is_active = ?", true).Count(&nodeCount).Error }) }()
	go func() { runQuery(func() error { return db.Model(&models.Order{}).Where("status = ?", "paid").Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue).Error }) }()
	go func() { runQuery(func() error { return db.Model(&models.User{}).Where("DATE(created_at) = ?", today).Count(&newUsersToday).Error }) }()

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
		"total_revenue":      totalRevenue,
	}

	cache.SetDashboardCache("admin_stats_overview", resultData, 60*time.Second)
	utils.Success(c, resultData)
}

// ==================== User Management ====================

func AdminListUsers(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	query := db.Model(&models.User{})
	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		// Also search by old subscription URL in reset history
		var resetUserIDs []uint
		db.Model(&models.SubscriptionReset{}).Where("old_subscription_url LIKE ? OR new_subscription_url LIKE ?", like, like).Distinct().Pluck("user_id", &resetUserIDs)
		// Search by current subscription URL
		var subUserIDs []uint
		db.Model(&models.Subscription{}).Where("subscription_url LIKE ?", like).Pluck("user_id", &subUserIDs)
		query = query.Where("username LIKE ? OR email LIKE ? OR id IN ? OR id IN ?", like, like, resetUserIDs, subUserIDs)
	}
	if status := c.Query("is_active"); status != "" {
		query = query.Where("is_active = ?", status == "true")
	}
	if isAdmin := c.Query("is_admin"); isAdmin != "" {
		query = query.Where("is_admin = ?", isAdmin == "true")
	}

	var total int64
	query.Count(&total)

	var users []models.User
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&users)

	// Enrich with level name and subscription fields needed by the edit dialog
	type UserItem struct {
		models.User
		LevelName  string     `json:"level_name"`
		ExpireTime *time.Time `json:"expire_time"`
		DeviceLimit int       `json:"device_limit"`
	}
	items := make([]UserItem, 0, len(users))
	// Pre-load all levels
	levelMap := make(map[uint]string)
	var levels []models.UserLevel
	db.Find(&levels)
	for _, l := range levels {
		levelMap[l.ID] = l.LevelName
	}

	subscriptionMap := make(map[uint]models.Subscription)
	if len(users) > 0 {
		userIDs := make([]uint, 0, len(users))
		for _, u := range users {
			userIDs = append(userIDs, u.ID)
		}
		var subscriptions []models.Subscription
		db.Where("user_id IN ?", userIDs).Find(&subscriptions)
		for _, sub := range subscriptions {
			subscriptionMap[sub.UserID] = sub
		}
	}

	for _, u := range users {
		item := UserItem{User: u}
		if u.UserLevelID != nil {
			item.LevelName = levelMap[*u.UserLevelID]
		}
		if sub, ok := subscriptionMap[u.ID]; ok {
			item.ExpireTime = &sub.ExpireTime
			item.DeviceLimit = sub.DeviceLimit
		}
		items = append(items, item)
	}

	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminGetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	var subscription models.Subscription
	db.Where("user_id = ?", id).First(&subscription)

	var orders []models.Order
	db.Where("user_id = ?", id).Order("created_at DESC").Limit(20).Find(&orders)

	var devices []models.Device
	db.Where("subscription_id = ?", subscription.ID).Order("last_access DESC").Find(&devices)

	var resets []models.SubscriptionReset
	db.Where("user_id = ?", id).Order("created_at DESC").Limit(20).Find(&resets)

	var balanceLogs []models.BalanceLog
	db.Where("user_id = ?", id).Order("created_at DESC").Limit(20).Find(&balanceLogs)

	var loginHistory []models.LoginHistory
	db.Where("user_id = ?", id).Order("login_time DESC").Limit(20).Find(&loginHistory)

	var rechargeRecords []models.RechargeRecord
	db.Where("user_id = ?", id).Order("created_at DESC").Limit(20).Find(&rechargeRecords)

	// Build subscription URLs
	baseURL := getSubscriptionBaseURL()
	subURLs := buildSubscriptionURLs(baseURL, subscription.SubscriptionURL)

	// Package name
	var packageName string
	if subscription.PackageID != nil {
		var pkg models.Package
		if db.Select("name").First(&pkg, *subscription.PackageID).Error == nil {
			packageName = pkg.Name
		}
	}

	utils.Success(c, gin.H{
		"user":              user,
		"subscription":      subscription,
		"subscription_urls": subURLs,
		"package_name":      packageName,
		"recent_orders":     orders,
		"devices":           devices,
		"resets":            resets,
		"balance_logs":      balanceLogs,
		"login_history":     loginHistory,
		"recharge_records":  rechargeRecords,
	})
}

func AdminUpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// Only allow updating safe fields
	allowed := map[string]bool{
		"username": true, "email": true, "is_active": true, "is_verified": true,
		"is_admin": true, "balance": true, "user_level_id": true, "notes": true,
	}
	updates := make(map[string]interface{})
	subscriptionUpdates := make(map[string]interface{})

	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		} else if k == "expire_time" || k == "device_limit" {
			// These fields belong to subscription table
			subscriptionUpdates[k] = v
		}
	}

	if len(updates) == 0 && len(subscriptionUpdates) == 0 {
		utils.BadRequest(c, "没有可更新的字段")
		return
	}

	// If balance is being changed, log it properly (before update)
	oldBalance := user.Balance
	var shouldLogBalance bool
	var newBalance float64
	if newBal, ok := updates["balance"]; ok {
		switch v := newBal.(type) {
		case float64:
			newBalance = v
		case int:
			newBalance = float64(v)
		default:
			newBalance = oldBalance
		}
		if newBalance != oldBalance {
			shouldLogBalance = true
		}
	}

	// Update user fields
	if len(updates) > 0 {
		if err := db.Model(&user).Updates(updates).Error; err != nil {
			utils.InternalError(c, "更新用户失败")
			return
		}
	}

	// Update subscription fields if provided
	if len(subscriptionUpdates) > 0 {
		var subscription models.Subscription
		if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err == nil {
			// 处理 expire_time 的时间格式转换
			if expireTimeStr, ok := subscriptionUpdates["expire_time"].(string); ok && expireTimeStr != "" {
				if expireTime, err := time.Parse(time.RFC3339, expireTimeStr); err == nil {
					subscriptionUpdates["expire_time"] = expireTime

					// 同步更新 status
					now := time.Now()
					if expireTime.After(now) {
						// 未过期
						if time.Until(expireTime) <= 7*24*time.Hour {
							subscriptionUpdates["status"] = "expiring"
						} else {
							subscriptionUpdates["status"] = "active"
						}
						// 确保 is_active 为 true
						subscriptionUpdates["is_active"] = true
					} else {
						// 已过期
						subscriptionUpdates["status"] = "expired"
					}
				}
			}

			if err := db.Model(&subscription).Updates(subscriptionUpdates).Error; err != nil {
				utils.InternalError(c, "更新订阅信息失败")
				return
			}
		}
	}

	// Create balance log after successful update
	if shouldLogBalance {
		diff := newBalance - oldBalance
		changeType := "admin_adjust"
		desc := fmt.Sprintf("管理员调整余额: %+.2f", diff)
		utils.CreateBalanceLogEntry(user.ID, changeType, diff, oldBalance, newBalance, nil, desc, c)
	}
	utils.CreateAuditLog(c, "update_user", "user", uint(id), fmt.Sprintf("更新用户: %s", user.Username))
	utils.Success(c, user)
}

func AdminDeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}
	// Send notification before deleting
	go services.NotifyUserDirect(user.Email, "account_deleted", nil)
	// Delete user and all related data in a transaction
	if err := db.Transaction(func(tx *gorm.DB) error {
		uid := user.ID
		// Clean up related records
		if err := tx.Where("user_id = ?", uid).Delete(&models.Subscription{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&models.Order{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&models.Device{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&models.InviteCode{}).Error; err != nil {
			return err
		}
		if err := tx.Where("inviter_id = ? OR invitee_id = ?", uid, uid).Delete(&models.InviteRelation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&models.Ticket{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&models.TicketReply{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&models.CheckIn{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&models.BalanceLog{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&models.MysteryBoxRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&models.CouponUsage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&user).Error
	}); err != nil {
		utils.InternalError(c, "删除用户失败")
		return
	}
	utils.CreateAuditLog(c, "delete_user", "user", uint(id), fmt.Sprintf("删除用户: %s (%s)", user.Username, user.Email))
	utils.SuccessMessage(c, "用户已删除")
}

func AdminDeleteUserDevice(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	deviceID, err := strconv.ParseUint(c.Param("deviceId"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的设备ID")
		return
	}
	db := database.GetDB()
	var device models.Device
	if err := db.First(&device, deviceID).Error; err != nil {
		utils.NotFound(c, "设备不存在")
		return
	}
	// Verify device belongs to this user's subscription
	var sub models.Subscription
	if err := db.Where("user_id = ?", userID).First(&sub).Error; err != nil {
		utils.NotFound(c, "用户订阅不存在")
		return
	}
	if device.SubscriptionID != sub.ID {
		utils.Forbidden(c, "设备不属于该用户")
		return
	}
	if err := db.Delete(&device).Error; err != nil {
		utils.InternalError(c, "删除设备失败")
		return
	}
	// Decrement current_devices
	if sub.CurrentDevices > 0 {
		if err := db.Model(&sub).UpdateColumn("current_devices", gorm.Expr("CASE WHEN current_devices > 0 THEN current_devices - 1 ELSE 0 END")).Error; err != nil {
			utils.InternalError(c, "更新设备计数失败")
			return
		}
	}
	utils.CreateAuditLog(c, "delete_device", "device", uint(deviceID), fmt.Sprintf("删除用户%d的设备%d", userID, deviceID))
	utils.SuccessMessage(c, "设备已删除")
}

func AdminToggleUserActive(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	currentID := c.GetUint("user_id")
	if id == uint64(currentID) {
		utils.BadRequest(c, "不能禁用自己的账号")
		return
	}
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}
	newStatus := !user.IsActive
	if !newStatus && user.IsAdmin {
		utils.BadRequest(c, "不能禁用其他管理员账号")
		return
	}
	if err := db.Model(&user).Update("is_active", newStatus).Error; err != nil {
		utils.InternalError(c, "更新用户状态失败")
		return
	}

	// Sync subscription status
	if newStatus {
		// Re-enable: set subscription status based on expire time
		var sub models.Subscription
		if db.Where("user_id = ?", id).First(&sub).Error == nil {
			updates := map[string]interface{}{"is_active": true}
			if sub.ExpireTime.After(time.Now()) {
				updates["status"] = "active"
			} else {
				updates["status"] = "expired"
			}
			if err := db.Model(&sub).Updates(updates).Error; err != nil {
				utils.InternalError(c, "同步订阅状态失败")
				return
			}
		}
	} else {
		// Disable: set subscription to disabled
		if err := db.Model(&models.Subscription{}).Where("user_id = ?", id).Updates(map[string]interface{}{
			"is_active": false,
			"status":    "disabled",
		}).Error; err != nil {
			utils.InternalError(c, "同步订阅状态失败")
			return
		}
	}

	// 通知用户账户状态变更
	if newStatus {
		go services.NotifyUser(user.ID, "account_enabled", nil)
	} else {
		go services.NotifyUser(user.ID, "account_disabled", nil)
	}
	action := "启用"
	if !newStatus {
		action = "禁用"
	}
	utils.CreateAuditLog(c, "toggle_user_active", "user", uint(id), fmt.Sprintf("%s用户: %s", action, user.Username))
	utils.Success(c, gin.H{"is_active": newStatus})
}

func AdminGetAbnormalUsers(c *gin.Context) {
	db := database.GetDB()

	// Filter by abnormal type if provided
	abnormalType := c.Query("type")

	type AbnormalUser struct {
		UserID       uint       `json:"user_id"`
		Username     string     `json:"username"`
		Email        string     `json:"email"`
		AbnormalType string     `json:"abnormal_type"`
		Details      string     `json:"details"`
		LastActive   *time.Time `json:"last_active"`
	}

	var abnormalUsers []AbnormalUser

	// 1. Users with too many subscription resets (more than 5)
	if abnormalType == "" || abnormalType == "excessive_resets" {
		type ResetCount struct {
			UserID uint
			Count  int64
		}
		var resetCounts []ResetCount
		db.Model(&models.SubscriptionReset{}).
			Select("user_id, COUNT(*) as count").
			Group("user_id").
			Having("COUNT(*) > ?", 5).
			Find(&resetCounts)

		for _, rc := range resetCounts {
			var user models.User
			if err := db.First(&user, rc.UserID).Error; err == nil {
				abnormalUsers = append(abnormalUsers, AbnormalUser{
					UserID:       user.ID,
					Username:     user.Username,
					Email:        user.Email,
					AbnormalType: "excessive_resets",
					Details:      strconv.FormatInt(rc.Count, 10) + " 次订阅重置",
					LastActive:   user.LastLogin,
				})
			}
		}
	}

	// 2. Users with too many devices (current_devices > device_limit)
	if abnormalType == "" || abnormalType == "device_limit_exceeded" {
		var subs []models.Subscription
		db.Where("current_devices > device_limit").Find(&subs)

		// 批量查询用户信息，避免 N+1 查询
		if len(subs) > 0 {
			userIDs := make([]uint, len(subs))
			for i, sub := range subs {
				userIDs[i] = sub.UserID
			}

			var users []models.User
			db.Where("id IN ?", userIDs).Find(&users)

			// 创建用户 ID 到用户的映射
			userMap := make(map[uint]models.User)
			for _, user := range users {
				userMap[user.ID] = user
			}

			// 使用映射避免重复查询
			for _, sub := range subs {
				if user, ok := userMap[sub.UserID]; ok {
					abnormalUsers = append(abnormalUsers, AbnormalUser{
						UserID:       user.ID,
						Username:     user.Username,
						Email:        user.Email,
						AbnormalType: "device_limit_exceeded",
						Details:      strconv.Itoa(sub.CurrentDevices) + "/" + strconv.Itoa(sub.DeviceLimit) + " 设备",
						LastActive:   user.LastLogin,
					})
				}
			}
		}
	}

	// 3. Users with suspicious login patterns (5+ different IPs in last 24 hours)
	if abnormalType == "" || abnormalType == "suspicious_logins" {
		type IPCount struct {
			UserID uint
			Count  int64
		}
		var ipCounts []IPCount
		yesterday := time.Now().Add(-24 * time.Hour)
		db.Model(&models.LoginHistory{}).
			Select("user_id, COUNT(DISTINCT ip_address) as count").
			Where("login_time > ? AND ip_address IS NOT NULL AND ip_address != ''", yesterday).
			Group("user_id").
			Having("COUNT(DISTINCT ip_address) >= ?", 5).
			Find(&ipCounts)

		// 批量查询用户信息，避免 N+1 查询
		if len(ipCounts) > 0 {
			userIDs := make([]uint, len(ipCounts))
			for i, ic := range ipCounts {
				userIDs[i] = ic.UserID
			}

			var users []models.User
			db.Where("id IN ?", userIDs).Find(&users)

			// 创建用户 ID 到用户的映射
			userMap := make(map[uint]models.User)
			for _, user := range users {
				userMap[user.ID] = user
			}

			// 使用映射避免重复查询
			for _, ic := range ipCounts {
				if user, ok := userMap[ic.UserID]; ok {
					abnormalUsers = append(abnormalUsers, AbnormalUser{
						UserID:       user.ID,
						Username:     user.Username,
						Email:        user.Email,
						AbnormalType: "suspicious_logins",
						Details:      strconv.FormatInt(ic.Count, 10) + " 个不同IP (24小时内)",
						LastActive:   user.LastLogin,
					})
				}
			}
		}
	}

	if abnormalUsers == nil {
		abnormalUsers = []AbnormalUser{}
	}
	utils.Success(c, gin.H{"users": abnormalUsers})
}

// ==================== Login As User ====================

func AdminLoginAsUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	accessToken, err := generateToken(user.ID, "access", 2*time.Hour)
	if err != nil {
		utils.InternalError(c, "生成访问令牌失败")
		return
	}
	refreshToken, err := generateToken(user.ID, "refresh", 24*time.Hour)
	if err != nil {
		utils.InternalError(c, "生成刷新令牌失败")
		return
	}

	utils.CreateAuditLog(c, "login_as_user", "user", uint(id), fmt.Sprintf("以用户身份登录: %s", user.Username))
	utils.Success(c, gin.H{
		"user": gin.H{
			"id": user.ID, "username": user.Username, "email": user.Email,
			"is_admin": user.IsAdmin, "nickname": user.Nickname, "avatar": user.Avatar,
			"balance": user.Balance, "theme": user.Theme,
		},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// ==================== Order Management ====================

func AdminListOrders(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	type AdminOrderItem struct {
		models.Order
		UserEmail     string  `json:"user_email"`
		OrderType     string  `json:"order_type"`
		OrderTypeText string  `json:"order_type_text"`
		OrderSummary  string  `json:"order_summary"`
		PackageName   string  `json:"package_name"`
		Devices       *int    `json:"devices,omitempty"`
		Months        *int    `json:"months,omitempty"`
		AddDevices    *int    `json:"add_devices,omitempty"`
		ExtendMonths  *int    `json:"extend_months,omitempty"`
		BalanceAmount *float64 `json:"balance_amount,omitempty"`
	}

	query := db.Table("orders").
		Select("orders.*, users.email as user_email").
		Joins("LEFT JOIN users ON users.id = orders.user_id")
	if status := c.Query("status"); status != "" {
		query = query.Where("orders.status = ?", status)
	}
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("orders.user_id = ?", userID)
	}
	if orderNo := c.Query("order_no"); orderNo != "" {
		query = query.Where("orders.order_no LIKE ?", "%"+orderNo+"%")
	}

	var total int64
	query.Count(&total)

	var items []AdminOrderItem
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Scan(&items)

	packageIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.PackageID > 0 {
			packageIDs = append(packageIDs, item.PackageID)
		}
	}

	pkgNameCache := make(map[uint]string)
	if len(packageIDs) > 0 {
		var packages []models.Package
		db.Select("id, name").Where("id IN ?", packageIDs).Find(&packages)
		for _, pkg := range packages {
			pkgNameCache[pkg.ID] = pkg.Name
		}
	}

	for i := range items {
		item := &items[i]
		item.OrderType = "package"
		item.OrderTypeText = "套餐订单"
		item.OrderSummary = "标准套餐"
		if name, ok := pkgNameCache[item.PackageID]; ok {
			item.PackageName = name
			item.OrderSummary = name
		}

		if item.PackageID == 0 && item.ExtraData != nil {
			var extra map[string]interface{}
			if json.Unmarshal([]byte(*item.ExtraData), &extra) == nil {
				switch extra["type"] {
				case "custom_package":
					item.OrderType = "custom_package"
					item.OrderTypeText = "充值信息"
					if devices, ok := extra["devices"].(float64); ok {
						v := int(devices)
						item.Devices = &v
					}
					if months, ok := extra["months"].(float64); ok {
						v := int(months)
						item.Months = &v
					}
					if item.Devices != nil && item.Months != nil {
						item.OrderSummary = fmt.Sprintf("%d设备 / %d个月", *item.Devices, *item.Months)
						item.PackageName = fmt.Sprintf("自定义套餐（%d设备/%d个月）", *item.Devices, *item.Months)
					}
				case "subscription_upgrade":
					item.OrderType = "subscription_upgrade"
					item.OrderTypeText = "设备升级"
					if addDevices, ok := extra["add_devices"].(float64); ok {
						v := int(addDevices)
						item.AddDevices = &v
					}
					if extendMonths, ok := extra["extend_months"].(float64); ok {
						v := int(extendMonths)
						item.ExtendMonths = &v
					}
					if item.AddDevices != nil {
						item.OrderSummary = fmt.Sprintf("+%d设备", *item.AddDevices)
						item.PackageName = item.OrderSummary
					}
					if item.AddDevices != nil && item.ExtendMonths != nil && *item.ExtendMonths > 0 {
						item.OrderSummary = fmt.Sprintf("+%d设备 / 续期%d个月", *item.AddDevices, *item.ExtendMonths)
						item.PackageName = item.OrderSummary
					}
				}
			}
		}
	}

	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminGetOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	utils.Success(c, order)
}

func AdminRefundOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	if order.Status != "paid" {
		utils.BadRequest(c, "只能退款已支付的订单")
		return
	}

	// Calculate refund amount
	refundAmount := order.Amount
	if order.FinalAmount != nil {
		refundAmount = *order.FinalAmount
	}

	// Try to refund via payment gateway if paid online
	var gatewayRefunded bool
	var txn models.PaymentTransaction
	if db.Where("order_id = ? AND status = ?", order.ID, "paid").First(&txn).Error == nil {
		// Has a successful payment transaction — try gateway refund
		if txn.ExternalTransactionID != nil && *txn.ExternalTransactionID != "" {
			// Check payment method to determine if it's direct Alipay
			var paymentMethod models.PaymentConfig
			if db.First(&paymentMethod, txn.PaymentMethodID).Error == nil && paymentMethod.PayType == "alipay" {
				// Direct Alipay refund
				if txn.TransactionID != nil && *txn.TransactionID != "" {
					if err := services.AlipayRefund(*txn.ExternalTransactionID, *txn.TransactionID, fmt.Sprintf("%.2f", refundAmount)); err != nil {
						utils.BadRequest(c, "支付宝退款失败: "+err.Error())
						return
					}
					gatewayRefunded = true
				}
			}
		}
	}

	tx := db.Begin()
	if tx.Error != nil {
		utils.InternalError(c, "创建事务失败")
		return
	}

	// If not refunded via gateway, refund to user balance
	if !gatewayRefunded {
		if err := tx.Model(&models.User{}).Where("id = ?", order.UserID).
			UpdateColumn("balance", gorm.Expr("balance + ?", refundAmount)).Error; err != nil {
			tx.Rollback()
			utils.InternalError(c, "退款失败")
			return
		}
	}

	// Update order status
	if err := tx.Model(&order).Update("status", "refunded").Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "退款失败")
		return
	}

	// Update payment transaction status
	if txn.ID > 0 {
		if err := tx.Model(&txn).Update("status", "refunded").Error; err != nil {
			tx.Rollback()
			utils.InternalError(c, "退款失败")
			return
		}
	}

	// Cancel/rollback the subscription that was activated by this order
	var sub models.Subscription
	if tx.Where("user_id = ?", order.UserID).First(&sub).Error == nil {
		shouldCancel := false
		if order.PackageID == 0 {
			// Custom package order — always cancel
			shouldCancel = true
		} else if sub.PackageID != nil && *sub.PackageID == int64(order.PackageID) {
			shouldCancel = true
		}
		if shouldCancel {
			if err := tx.Model(&sub).Updates(map[string]interface{}{
				"is_active": false,
				"status":    "cancelled",
			}).Error; err != nil {
				tx.Rollback()
				utils.InternalError(c, "退款失败")
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		utils.InternalError(c, "退款事务提交失败")
		return
	}

	// Log
	refundMethod := "余额"
	if gatewayRefunded {
		refundMethod = "原路退回"
	}
	var refundUser models.User
	if db.First(&refundUser, order.UserID).Error == nil {
		desc := fmt.Sprintf("管理员退款订单: %s (%s)", order.OrderNo, refundMethod)
		if !gatewayRefunded {
			utils.CreateBalanceLogEntry(order.UserID, "refund", refundAmount, refundUser.Balance-refundAmount, refundUser.Balance, func() *uint { id := uint(order.ID); return &id }(), desc, c)
		}
	}
	utils.CreateAuditLog(c, "refund_order", "order", uint(id), fmt.Sprintf("退款订单: %s, 金额: %.2f, 方式: %s", order.OrderNo, refundAmount, refundMethod))
	utils.SuccessMessage(c, fmt.Sprintf("退款成功（%s）", refundMethod))
}

func AdminCancelOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	if order.Status != "pending" {
		utils.BadRequest(c, "只能取消待支付的订单")
		return
	}

	if err := db.Model(&order).Update("status", "cancelled").Error; err != nil {
		utils.InternalError(c, "取消订单失败")
		return
	}

	utils.CreateAuditLog(c, "cancel_order", "order", uint(id), fmt.Sprintf("取消订单: %s", order.OrderNo))
	utils.SuccessMessage(c, "订单已取消")
}

func AdminCompleteOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	if order.Status != "paid" {
		utils.BadRequest(c, "只能完成已支付的订单")
		return
	}

	if err := db.Model(&order).Update("status", "completed").Error; err != nil {
		utils.InternalError(c, "完成订单失败")
		return
	}

	utils.CreateAuditLog(c, "complete_order", "order", uint(id), fmt.Sprintf("完成订单: %s", order.OrderNo))
	utils.SuccessMessage(c, "订单已完成")
}

func AdminDeleteOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	if order.Status != "cancelled" && order.Status != "refunded" {
		utils.BadRequest(c, "只能删除已取消或已退款的订单")
		return
	}

	if err := db.Delete(&order).Error; err != nil {
		utils.InternalError(c, "删除订单失败")
		return
	}

	utils.CreateAuditLog(c, "delete_order", "order", uint(id), fmt.Sprintf("删除订单: %s", order.OrderNo))
	utils.SuccessMessage(c, "订单已删除")
}

// ==================== Package Management ====================

func AdminListPackages(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	query := db.Model(&models.Package{})
	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	var total int64
	query.Count(&total)

	var packages []models.Package
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&packages)

	utils.SuccessPage(c, packages, total, p.Page, p.PageSize)
}

func AdminCreatePackage(c *gin.Context) {
	var pkg models.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := database.GetDB().Create(&pkg).Error; err != nil {
		utils.InternalError(c, "创建套餐失败")
		return
	}
	utils.CreateAuditLog(c, "create_package", "package", pkg.ID, fmt.Sprintf("创建套餐: %s", pkg.Name))
	utils.Success(c, pkg)
}

func AdminUpdatePackage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的套餐ID")
		return
	}
	db := database.GetDB()
	var pkg models.Package
	if err := db.First(&pkg, id).Error; err != nil {
		utils.NotFound(c, "套餐不存在")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	allowed := map[string]bool{
		"name": true, "description": true, "price": true, "duration_days": true,
		"device_limit": true, "is_active": true, "sort_order": true, "features": true,
		"original_price": true, "discount_text": true, "badge": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	if err := db.Model(&pkg).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新套餐失败")
		return
	}
	utils.CreateAuditLog(c, "update_package", "package", uint(id), fmt.Sprintf("更新套餐: %s", pkg.Name))
	utils.Success(c, pkg)
}

func AdminDeletePackage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的套餐ID")
		return
	}
	if err := database.GetDB().Delete(&models.Package{}, id).Error; err != nil {
		utils.InternalError(c, "删除套餐失败")
		return
	}
	utils.CreateAuditLog(c, "delete_package", "package", uint(id), "删除套餐")
	utils.SuccessMessage(c, "套餐已删除")
}

// ==================== Node Management ====================

func AdminListNodes(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	query := db.Model(&models.Node{})
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if region := c.Query("region"); region != "" {
		query = query.Where("region = ?", region)
	}
	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR region LIKE ? OR type LIKE ? OR description LIKE ?", like, like, like, like)
	}

	var total int64
	query.Count(&total)

	var nodes []models.Node
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&nodes)

	utils.SuccessPage(c, nodes, total, p.Page, p.PageSize)
}

func AdminCreateNode(c *gin.Context) {
	var req struct {
		Name          string  `json:"name" binding:"required"`
		Region        string  `json:"region"`
		Type          string  `json:"type"`
		Status        string  `json:"status"`
		Description   *string `json:"description"`
		Config        *string `json:"config"`
		IsRecommended bool    `json:"is_recommended"`
		OrderIndex    int     `json:"order_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	node := models.Node{
		Name: req.Name, Region: req.Region, Type: req.Type, Status: req.Status,
		Description: req.Description, Config: req.Config, IsRecommended: req.IsRecommended,
		OrderIndex: req.OrderIndex, IsManual: true,
	}
	if err := database.GetDB().Create(&node).Error; err != nil {
		utils.InternalError(c, "创建节点失败")
		return
	}
	utils.CreateAuditLog(c, "create_node", "node", node.ID, fmt.Sprintf("创建节点: %s", node.Name))
	cache.ClearAllSubscriptionCache()
	utils.Success(c, node)
}

func AdminUpdateNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的节点ID")
		return
	}
	db := database.GetDB()
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		utils.NotFound(c, "节点不存在")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	allowed := map[string]bool{
		"name": true, "region": true, "type": true, "status": true, "description": true,
		"config": true, "is_recommended": true, "is_active": true, "is_manual": true,
		"order_index": true, "source_index": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	if err := db.Model(&node).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新节点失败")
		return
	}
	utils.CreateAuditLog(c, "update_node", "node", uint(id), fmt.Sprintf("更新节点: %s", node.Name))
	cache.ClearAllSubscriptionCache()
	utils.Success(c, node)
}

func AdminDeleteNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的节点ID")
		return
	}
	if err := database.GetDB().Delete(&models.Node{}, id).Error; err != nil {
		utils.InternalError(c, "删除节点失败")
		return
	}
	utils.CreateAuditLog(c, "delete_node", "node", uint(id), "删除节点")
	cache.ClearAllSubscriptionCache()
	utils.SuccessMessage(c, "节点已删除")
}

func AdminImportNodes(c *gin.Context) {
	var req struct {
		Type  string `json:"type" binding:"required"`
		URL   string `json:"url"`
		Links string `json:"links"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	var content string
	var err error

	switch req.Type {
	case "subscription":
		if req.URL == "" {
			utils.BadRequest(c, "订阅URL不能为空")
			return
		}
		content, err = services.FetchSubscriptionContent(req.URL)
		if err != nil {
			utils.BadRequest(c, "获取订阅内容失败: "+err.Error())
			return
		}
	case "links":
		if req.Links == "" {
			utils.BadRequest(c, "节点链接不能为空")
			return
		}
		content = req.Links
	default:
		utils.BadRequest(c, "不支持的导入类型")
		return
	}

	nodes, err := services.ParseSubscriptionContent(content)
	if err != nil {
		utils.BadRequest(c, "解析节点失败: "+err.Error())
		return
	}

	if len(nodes) == 0 {
		utils.BadRequest(c, "未找到有效的节点")
		return
	}

	db := database.GetDB()
	successCount := 0
	for _, node := range nodes {
		node.IsManual = true // 管理员手动导入的节点，自动更新时不会被删除
		if err := db.Create(&node).Error; err == nil {
			successCount++
		}
	}

	cache.ClearAllSubscriptionCache()
	utils.Success(c, gin.H{
		"total":   len(nodes),
		"success": successCount,
		"message": "导入完成",
	})
}

func AdminTestNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的节点ID")
		return
	}
	db := database.GetDB()
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		utils.NotFound(c, "节点不存在")
		return
	}
	if node.Config == nil || *node.Config == "" {
		utils.BadRequest(c, "节点无配置信息")
		return
	}

	latency, reachable := testNodeConnectivity(*node.Config)
	now := time.Now()
	status := "offline"
	if reachable {
		status = "online"
	}
	if err := db.Model(&node).Updates(map[string]interface{}{
		"status": status, "latency": latency, "last_test": &now,
	}).Error; err != nil {
		utils.InternalError(c, "更新节点测试结果失败")
		return
	}

	utils.Success(c, gin.H{
		"node_id":   node.ID,
		"name":      node.Name,
		"status":    status,
		"latency":   latency,
		"reachable": reachable,
	})
}

// ==================== Custom Node Management ====================

func AdminListCustomNodes(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)
	query := db.Model(&models.CustomNode{})

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		var matchedNodeIDs []uint
		db.Model(&models.UserCustomNode{}).
			Joins("JOIN users ON users.id = user_custom_nodes.user_id").
			Where("users.email LIKE ? OR users.username LIKE ?", like, like).
			Distinct().
			Pluck("user_custom_nodes.custom_node_id", &matchedNodeIDs)

		query = query.Where(
			db.Where("custom_nodes.name LIKE ? OR custom_nodes.display_name LIKE ? OR custom_nodes.domain LIKE ? OR CAST(custom_nodes.port AS CHAR) LIKE ?", like, like, like, like).
				Or("custom_nodes.id IN ?", matchedNodeIDs),
		)
	}

	var total int64
	query.Count(&total)

	var nodes []models.CustomNode
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&nodes)

	utils.SuccessPage(c, nodes, total, p.Page, p.PageSize)
}

func AdminCreateCustomNode(c *gin.Context) {
	var req struct {
		Name             string     `json:"name" binding:"required"`
		DisplayName      string     `json:"display_name"`
		Protocol         string     `json:"protocol"`
		Domain           string     `json:"domain"`
		Port             int        `json:"port"`
		Config           string     `json:"config"`
		Status           string     `json:"status"`
		ExpireTime       *time.Time `json:"expire_time"`
		FollowUserExpire bool       `json:"follow_user_expire"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	node := models.CustomNode{
		Name:             req.Name,
		DisplayName:      req.DisplayName,
		Domain:           req.Domain,
		Port:             req.Port,
		Protocol:         req.Protocol,
		Status:           req.Status,
		Config:           req.Config,
		ExpireTime:       req.ExpireTime,
		FollowUserExpire: req.FollowUserExpire,
	}
	if err := database.GetDB().Create(&node).Error; err != nil {
		utils.InternalError(c, "创建专线节点失败")
		return
	}
	cache.ClearAllSubscriptionCache()
	utils.Success(c, node)
}

func AdminUpdateCustomNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}
	db := database.GetDB()
	var node models.CustomNode
	if err := db.First(&node, id).Error; err != nil {
		utils.NotFound(c, "专线节点不存在")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	allowed := map[string]bool{
		"name": true, "display_name": true, "protocol": true, "domain": true, "port": true,
		"config": true, "status": true, "is_active": true, "expire_time": true,
		"follow_user_expire": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	if err := db.Model(&node).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新专线节点失败")
		return
	}
	cache.ClearAllSubscriptionCache()
	utils.Success(c, node)
}

func AdminDeleteCustomNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}
	db := database.GetDB()
	// Remove user assignments first
	if err := db.Where("custom_node_id = ?", id).Delete(&models.UserCustomNode{}).Error; err != nil {
		utils.InternalError(c, "删除专线节点分配关系失败")
		return
	}
	if err := db.Delete(&models.CustomNode{}, id).Error; err != nil {
		utils.InternalError(c, "删除专线节点失败")
		return
	}
	cache.ClearAllSubscriptionCache()
	utils.SuccessMessage(c, "专线节点已删除")
}

func AdminAssignCustomNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}
	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if len(req.UserIDs) == 0 {
		utils.BadRequest(c, "请选择要分配的用户")
		return
	}
	if err := replaceCustomNodeAssignments(database.GetDB(), uint(id), req.UserIDs); err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.CreateAuditLog(c, "assign_custom_node", "custom_node", uint(id), fmt.Sprintf("分配专线节点给 %d 个用户", len(req.UserIDs)))
	cache.ClearAllSubscriptionCache()
	utils.SuccessMessage(c, "分配成功")
}

func AdminBatchAssignCustomNodes(c *gin.Context) {
	var req struct {
		IDs     []uint `json:"ids" binding:"required"`
		UserIDs []uint `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择要分配的专线节点")
		return
	}
	if len(req.UserIDs) == 0 {
		utils.BadRequest(c, "请选择要分配的用户")
		return
	}

	db := database.GetDB()
	uniqueNodeIDs := uniqueUintSlice(req.IDs)
	uniqueUserIDs := uniqueUintSlice(req.UserIDs)
	successCount := 0

	for _, nodeID := range uniqueNodeIDs {
		if err := replaceCustomNodeAssignments(db, nodeID, uniqueUserIDs); err == nil {
			successCount++
		}
	}

	if successCount == 0 {
		utils.InternalError(c, "批量分配失败")
		return
	}

	utils.CreateAuditLog(c, "batch_assign_custom_node", "custom_node", 0, fmt.Sprintf("批量分配 %d 个专线节点给 %d 个用户", len(uniqueNodeIDs), len(uniqueUserIDs)))
	cache.ClearAllSubscriptionCache()
	utils.Success(c, gin.H{
		"success": successCount,
		"total":   len(uniqueNodeIDs),
		"message": "批量分配成功",
	})
}

func replaceCustomNodeAssignments(db *gorm.DB, nodeID uint, userIDs []uint) error {
	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		return fmt.Errorf("专线节点不存在")
	}

	uniqueUserIDs := uniqueUintSlice(userIDs)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("custom_node_id = ?", nodeID).Delete(&models.UserCustomNode{}).Error; err != nil {
			return fmt.Errorf("清理分配关系失败")
		}
		for _, uid := range uniqueUserIDs {
			assignment := models.UserCustomNode{UserID: uid, CustomNodeID: nodeID}
			if err := tx.Create(&assignment).Error; err != nil {
				return fmt.Errorf("分配专线节点失败")
			}
		}
		return nil
	})
}

func uniqueUintSlice(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func AdminImportCustomNodeLinks(c *gin.Context) {
	var req struct {
		Links string `json:"links" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	nodes, err := services.ParseSubscriptionContent(req.Links)
	if err != nil {
		utils.BadRequest(c, "解析节点失败: "+err.Error())
		return
	}
	if len(nodes) == 0 {
		utils.BadRequest(c, "未找到有效的节点")
		return
	}

	db := database.GetDB()
	successCount := 0
	for _, node := range nodes {
		domain := ""
		port := 443
		if node.Config != nil && *node.Config != "" {
			if extractedDomain, extractedPort, extractErr := services.ExtractDomainPortFromNodeLink(*node.Config); extractErr == nil {
				domain = extractedDomain
				if extractedPort > 0 {
					port = extractedPort
				}
			}
		}
		customNode := models.CustomNode{
			Name:        node.Name,
			DisplayName: node.Name,
			Protocol:    node.Type,
			Domain:      domain,
			Port:        port,
			Config:      "",
			IsActive:    true,
		}
		if node.Config != nil {
			customNode.Config = *node.Config
		}
		if err := db.Create(&customNode).Error; err == nil {
			successCount++
		}
	}

	cache.ClearAllSubscriptionCache()
	utils.Success(c, gin.H{
		"total":   len(nodes),
		"success": successCount,
		"message": "导入完成",
	})
}

func AdminBatchDeleteCustomNodes(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择要删除的节点")
		return
	}

	db := database.GetDB()
	if err := db.Where("custom_node_id IN ?", req.IDs).Delete(&models.UserCustomNode{}).Error; err != nil {
		utils.InternalError(c, "批量删除分配关系失败")
		return
	}
	result := db.Where("id IN ?", req.IDs).Delete(&models.CustomNode{})
	cache.ClearAllSubscriptionCache()
	utils.Success(c, gin.H{
		"deleted": result.RowsAffected,
		"message": "批量删除完成",
	})
}

func AdminGetCustomNodeLink(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}
	db := database.GetDB()
	var node models.CustomNode
	if err := db.First(&node, id).Error; err != nil {
		utils.NotFound(c, "专线节点不存在")
		return
	}
	utils.Success(c, gin.H{
		"link":     node.Config,
		"name":     node.DisplayName,
		"protocol": node.Protocol,
	})
}

func AdminGetCustomNodeUsers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}
	db := database.GetDB()
	var assignments []models.UserCustomNode
	db.Where("custom_node_id = ?", id).Find(&assignments)

	var userIDs []uint
	for _, a := range assignments {
		userIDs = append(userIDs, a.UserID)
	}

	var users []models.User
	if len(userIDs) > 0 {
		db.Where("id IN ?", userIDs).Select("id, username, email").Find(&users)
	}

	utils.Success(c, gin.H{
		"user_ids": userIDs,
		"users":    users,
	})
}

// ==================== Subscription Management ====================

func AdminListSubscriptions(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	query := db.Model(&models.Subscription{})
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if search := c.Query("search"); search != "" {
		// Search by user email, username, notes, or subscription URL (current + old)
		like := "%" + search + "%"
		var userIDs []uint
		db.Model(&models.User{}).Where("email LIKE ? OR username LIKE ? OR notes LIKE ? OR CAST(id AS CHAR) = ?",
			like, like, like, search).Pluck("id", &userIDs)
		// Also match current subscription URL
		var subIDs []uint
		db.Model(&models.Subscription{}).Where("subscription_url LIKE ?", like).Pluck("id", &subIDs)
		// Also match old subscription URLs from reset history
		var resetSubIDs []uint
		db.Model(&models.SubscriptionReset{}).Where("old_subscription_url LIKE ? OR new_subscription_url LIKE ?", like, like).Distinct().Pluck("subscription_id", &resetSubIDs)
		if len(userIDs) > 0 || len(subIDs) > 0 || len(resetSubIDs) > 0 {
			query = query.Where("user_id IN ? OR id IN ? OR id IN ?", userIDs, subIDs, resetSubIDs)
		} else {
			query = query.Where("1 = 0") // no match
		}
	}

	var total int64
	query.Count(&total)

	var subs []models.Subscription
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&subs)

	// Enrich with user email, package name, and subscription URLs for QR code
	baseURL := getSubscriptionBaseURL()
	type SubItem struct {
		models.Subscription
		UserEmail    string  `json:"user_email"`
		Username     string  `json:"username"`
		PackageName  string  `json:"package_name"`
		UserNotes    *string `json:"user_notes"`
		UniversalURL string  `json:"universal_url"`
		ClashURL     string  `json:"clash_url"`
	}
	items := make([]SubItem, 0, len(subs))
	for _, sub := range subs {
		item := SubItem{Subscription: sub}
		subURLs := buildSubscriptionURLs(baseURL, sub.SubscriptionURL)
		item.UniversalURL, _ = subURLs["universal_url"].(string)
		item.ClashURL, _ = subURLs["clash_url"].(string)
		var user models.User
		if db.Select("email, username, notes").First(&user, sub.UserID).Error == nil {
			item.UserEmail = user.Email
			item.Username = user.Username
			item.UserNotes = user.Notes
		}
		if sub.PackageID != nil {
			var pkg models.Package
			if db.Select("name").First(&pkg, *sub.PackageID).Error == nil {
				item.PackageName = pkg.Name
			}
		}
		// 仍在有效期内时，以到期时间为准纠正 status，避免显示“已过期”
		if sub.IsActive && sub.ExpireTime.After(time.Now()) {
			if time.Until(sub.ExpireTime) <= 7*24*time.Hour {
				item.Status = "expiring"
			} else {
				item.Status = "active"
			}
		}
		items = append(items, item)
	}

	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminUpdateUserNotes(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}
	if err := db.Model(&user).Update("notes", req.Notes).Error; err != nil {
		utils.InternalError(c, "更新备注失败")
		return
	}
	utils.SuccessMessage(c, "备注已更新")
}

func AdminGetSubscription(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订阅ID")
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		utils.NotFound(c, "订阅不存在")
		return
	}

	var devices []models.Device
	db.Where("subscription_id = ?", sub.ID).Find(&devices)

	result := gin.H{
		"id":               sub.ID,
		"user_id":          sub.UserID,
		"package_id":       sub.PackageID,
		"subscription_url": sub.SubscriptionURL,
		"device_limit":     sub.DeviceLimit,
		"current_devices":  sub.CurrentDevices,
		"universal_count":  sub.UniversalCount,
		"clash_count":      sub.ClashCount,
		"is_active":        sub.IsActive,
		"status":           sub.Status,
		"expire_time":      sub.ExpireTime,
		"created_at":       sub.CreatedAt,
		"updated_at":       sub.UpdatedAt,
		"devices":          devices,
	}

	// Build full subscription URLs
	baseURL := getSubscriptionBaseURL()
	for key, value := range buildSubscriptionURLs(baseURL, sub.SubscriptionURL) {
		result[key] = value
	}

	var user models.User
	if db.First(&user, sub.UserID).Error == nil {
		result["user_email"] = user.Email
		result["username"] = user.Username
		result["user_balance"] = user.Balance
		result["user_is_active"] = user.IsActive
		result["user_is_admin"] = user.IsAdmin
		result["user_created_at"] = user.CreatedAt
		result["user_last_login"] = user.LastLogin
		if user.UserLevelID != nil {
			var level models.UserLevel
			if db.Select("level_name").First(&level, *user.UserLevelID).Error == nil {
				result["user_level_name"] = level.LevelName
			}
		}
	}
	if sub.PackageID != nil {
		var pkg models.Package
		if db.Select("name").First(&pkg, *sub.PackageID).Error == nil {
			result["package_name"] = pkg.Name
		}
	}

	// Rich user data (same as AdminGetUser)
	var orders []models.Order
	db.Where("user_id = ?", sub.UserID).Order("created_at DESC").Limit(20).Find(&orders)
	result["recent_orders"] = orders

	var balanceLogs []models.BalanceLog
	db.Where("user_id = ?", sub.UserID).Order("created_at DESC").Limit(20).Find(&balanceLogs)
	result["balance_logs"] = balanceLogs

	var loginHistory []models.LoginHistory
	db.Where("user_id = ?", sub.UserID).Order("login_time DESC").Limit(20).Find(&loginHistory)
	result["login_history"] = loginHistory

	var resets []models.SubscriptionReset
	db.Where("user_id = ?", sub.UserID).Order("created_at DESC").Limit(20).Find(&resets)
	result["resets"] = resets

	var rechargeRecords []models.RechargeRecord
	db.Where("user_id = ?", sub.UserID).Order("created_at DESC").Limit(20).Find(&rechargeRecords)
	result["recharge_records"] = rechargeRecords

	utils.Success(c, result)
}

func AdminResetSubscription(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订阅ID")
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		utils.NotFound(c, "订阅不存在")
		return
	}

	oldURL := sub.SubscriptionURL
	newURL := utils.GenerateRandomString(64)
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("subscription_id = ?", sub.ID).Delete(&models.Device{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&sub).Updates(map[string]interface{}{
			"subscription_url": newURL,
			"current_devices":  0,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&models.SubscriptionReset{
			UserID:             sub.UserID,
			SubscriptionID:     sub.ID,
			ResetType:          "admin_reset",
			Reason:             "管理员重置",
			OldSubscriptionURL: &oldURL,
			NewSubscriptionURL: &newURL,
			DeviceCountBefore:  sub.CurrentDevices,
			DeviceCountAfter:   0,
		}).Error
	}); err != nil {
		utils.InternalError(c, "重置订阅失败")
		return
	}

	// 通知用户订阅已重置
	go services.NotifyUser(sub.UserID, "subscription_reset", map[string]string{"reset_by": "管理员"})

	adminID := c.GetUint("user_id")
	utils.CreateSubscriptionLog(sub.ID, sub.UserID, "reset", "admin", &adminID, "管理员重置订阅", nil, nil)
	utils.CreateAuditLog(c, "reset_subscription", "subscription", uint(id), fmt.Sprintf("重置订阅 (用户ID: %d)", sub.UserID))
	utils.Success(c, gin.H{"new_subscription_url": newURL})
}

func AdminExtendSubscription(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订阅ID")
		return
	}
	var req struct {
		Days int `json:"days" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		utils.NotFound(c, "订阅不存在")
		return
	}

	newExpire := sub.ExpireTime
	if newExpire.Before(time.Now()) {
		newExpire = time.Now()
	}
	newExpire = newExpire.AddDate(0, 0, req.Days)
	if err := db.Model(&sub).Updates(map[string]interface{}{
		"expire_time": newExpire,
		"is_active":   true,
		"status":      "active",
	}).Error; err != nil {
		utils.InternalError(c, "延长订阅失败")
		return
	}

	adminID := c.GetUint("user_id")
	utils.CreateSubscriptionLog(sub.ID, sub.UserID, "extend", "admin", &adminID, fmt.Sprintf("管理员延长订阅 %d 天", req.Days), nil, nil)
	utils.CreateAuditLog(c, "extend_subscription", "subscription", uint(id), fmt.Sprintf("延长订阅 %d 天 (用户ID: %d)", req.Days, sub.UserID))
	utils.Success(c, gin.H{"new_expire_time": newExpire})
}

// ==================== Coupon Management ====================

func AdminListCoupons(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	var total int64
	db.Model(&models.Coupon{}).Count(&total)

	var coupons []models.Coupon
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&coupons)

	utils.SuccessPage(c, coupons, total, p.Page, p.PageSize)
}

func AdminCreateCoupon(c *gin.Context) {
	var req struct {
		Code                 string   `json:"code" binding:"required"`
		Name                 string   `json:"name"`
		Description          string   `json:"description"`
		Type                 string   `json:"type" binding:"required"`
		DiscountValue        float64  `json:"discount_value"`
		MaxDiscount          *float64 `json:"max_discount"`
		MinAmount            float64  `json:"min_amount"`
		ValidFrom            string   `json:"valid_from" binding:"required"`
		ValidUntil           string   `json:"valid_until" binding:"required"`
		TotalQuantity        *int64   `json:"total_quantity"`
		MaxUsesPerUser       int      `json:"max_uses_per_user"`
		Status               string   `json:"status"`
		ApplicablePackageIDs string   `json:"applicable_package_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	validFrom, err := time.Parse(time.RFC3339, req.ValidFrom)
	if err != nil {
		validFrom, err = time.Parse("2006-01-02", req.ValidFrom)
		if err != nil {
			utils.BadRequest(c, "valid_from 日期格式错误")
			return
		}
	}
	validUntil, err := time.Parse(time.RFC3339, req.ValidUntil)
	if err != nil {
		validUntil, err = time.Parse("2006-01-02", req.ValidUntil)
		if err != nil {
			utils.BadRequest(c, "valid_until 日期格式错误")
			return
		}
	}
	adminID := c.GetUint("user_id")
	adminIDInt64 := int64(adminID)
	coupon := models.Coupon{
		Code: req.Code, Name: req.Name, Description: req.Description, Type: req.Type,
		DiscountValue: req.DiscountValue, MaxDiscount: req.MaxDiscount, MinAmount: &req.MinAmount,
		ValidFrom: validFrom, ValidUntil: validUntil, TotalQuantity: req.TotalQuantity,
		MaxUsesPerUser: req.MaxUsesPerUser, Status: req.Status, CreatedBy: &adminIDInt64,
		ApplicablePackages: req.ApplicablePackageIDs,
	}
	if coupon.Status == "" {
		coupon.Status = "active"
	}
	if coupon.MaxUsesPerUser == 0 {
		coupon.MaxUsesPerUser = 1
	}

	if err := database.GetDB().Create(&coupon).Error; err != nil {
		utils.InternalError(c, "创建优惠券失败")
		return
	}
	utils.Success(c, coupon)
}

func AdminUpdateCoupon(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	db := database.GetDB()
	var coupon models.Coupon
	if err := db.First(&coupon, id).Error; err != nil {
		utils.NotFound(c, "优惠券不存在")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	allowed := map[string]bool{
		"name": true, "description": true, "type": true, "discount_value": true,
		"min_amount": true, "valid_from": true, "valid_until": true,
		"total_quantity": true, "max_uses_per_user": true, "status": true,
		"applicable_package_ids": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	if err := db.Model(&coupon).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新优惠券失败")
		return
	}
	db.First(&coupon, id)
	utils.Success(c, coupon)
}

func AdminDeleteCoupon(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的优惠券ID")
		return
	}
	if err := database.GetDB().Delete(&models.Coupon{}, id).Error; err != nil {
		utils.InternalError(c, "删除优惠券失败")
		return
	}
	utils.SuccessMessage(c, "优惠券已删除")
}

// ==================== Ticket Management ====================

func AdminListTickets(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	query := db.Model(&models.Ticket{})
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	var total int64
	query.Count(&total)

	var tickets []models.Ticket
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&tickets)

	utils.SuccessPage(c, tickets, total, p.Page, p.PageSize)
}

func AdminGetTicket(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的工单ID")
		return
	}
	db := database.GetDB()
	var ticket models.Ticket
	if err := db.First(&ticket, id).Error; err != nil {
		utils.NotFound(c, "工单不存在")
		return
	}
	var replies []models.TicketReply
	db.Where("ticket_id = ?", ticket.ID).Order("created_at ASC").Find(&replies)

	utils.Success(c, gin.H{"ticket": ticket, "replies": replies})
}

func AdminUpdateTicket(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的工单ID")
		return
	}
	db := database.GetDB()
	var ticket models.Ticket
	if err := db.First(&ticket, id).Error; err != nil {
		utils.NotFound(c, "工单不存在")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	allowed := map[string]bool{
		"status": true, "priority": true, "assigned_to": true, "admin_notes": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	if err := db.Model(&ticket).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新工单失败")
		return
	}
	utils.Success(c, ticket)
}

func AdminReplyTicket(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	adminID := c.GetUint("user_id")
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	db := database.GetDB()
	reply := models.TicketReply{TicketID: uint(id), UserID: adminID, Content: req.Content, IsAdmin: true}
	if err := db.Create(&reply).Error; err != nil {
		utils.InternalError(c, "回复工单失败")
		return
	}
	if err := db.Model(&models.Ticket{}).Where("id = ?", id).Update("status", "processing").Error; err != nil {
		utils.InternalError(c, "更新工单状态失败")
		return
	}
	utils.Success(c, reply)
}

func AdminListUserLevels(c *gin.Context) {
	var levels []models.UserLevel
	database.GetDB().Order("level_order ASC").Find(&levels)
	utils.Success(c, levels)
}

func AdminCreateUserLevel(c *gin.Context) {
	var req struct {
		LevelName      string  `json:"level_name" binding:"required"`
		LevelOrder     int     `json:"level_order"`
		DiscountRate   float64 `json:"discount_rate"`
		MinConsumption float64 `json:"min_consumption"`
		Benefits       *string `json:"benefits"`
		IconURL        *string `json:"icon_url"`
		Color          string  `json:"color"`
		IsActive       *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	level := models.UserLevel{
		LevelName: req.LevelName, LevelOrder: req.LevelOrder, DiscountRate: req.DiscountRate,
		MinConsumption: req.MinConsumption, Benefits: req.Benefits, IconURL: req.IconURL,
		Color: req.Color,
	}
	if req.IsActive != nil {
		level.IsActive = *req.IsActive
	} else {
		level.IsActive = true
	}
	if err := database.GetDB().Create(&level).Error; err != nil {
		utils.InternalError(c, "创建用户等级失败")
		return
	}
	utils.Success(c, level)
}

func AdminUpdateUserLevel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的等级ID")
		return
	}
	db := database.GetDB()
	var level models.UserLevel
	if err := db.First(&level, id).Error; err != nil {
		utils.NotFound(c, "等级不存在")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	allowed := map[string]bool{
		"name": true, "level_order": true, "discount_rate": true,
		"description": true, "required_exp": true, "is_active": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	if err := db.Model(&level).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新用户等级失败")
		return
	}
	utils.Success(c, level)
}

func AdminDeleteUserLevel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	if err := database.GetDB().Delete(&models.UserLevel{}, id).Error; err != nil {
		utils.InternalError(c, "删除用户等级失败")
		return
	}
	utils.SuccessMessage(c, "等级已删除")
}

func AdminListRedeemCodes(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.RedeemCode
	var total int64
	db := database.GetDB().Model(&models.RedeemCode{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminCreateRedeemCodes(c *gin.Context) {
	var req struct {
		Code      string  `json:"code"`
		Name      string  `json:"name"`
		Type      string  `json:"type" binding:"required"`
		Value     float64 `json:"value" binding:"required"`
		PackageID *uint   `json:"package_id"`
		Quantity  int     `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	adminID := c.GetUint("user_id")
	db := database.GetDB()
	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}
	name := req.Name
	if name == "" {
		name = req.Type + " 卡密"
	}
	var codes []models.RedeemCode
	for i := 0; i < qty; i++ {
		code := req.Code
		if code == "" || qty > 1 {
			code = utils.GenerateRandomString(16)
		}
		rc := models.RedeemCode{
			Code:      code,
			Name:      name,
			Type:      req.Type,
			Value:     req.Value,
			PackageID: req.PackageID,
			Status:    "unused",
			CreatedBy: adminID,
		}
		if err := db.Create(&rc).Error; err != nil {
			utils.InternalError(c, "创建卡密失败")
			return
		}
		codes = append(codes, rc)
	}
	codeStrings := make([]string, len(codes))
	for i, rc := range codes {
		codeStrings[i] = rc.Code
	}
	utils.Success(c, gin.H{"codes": codeStrings})
}

func AdminDeleteRedeemCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	if err := database.GetDB().Delete(&models.RedeemCode{}, id).Error; err != nil {
		utils.InternalError(c, "删除卡密失败")
		return
	}
	utils.SuccessMessage(c, "卡密已删除")
}

func AdminListEmailQueue(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.EmailQueue
	var total int64
	db := database.GetDB().Model(&models.EmailQueue{})
	status := c.Query("status")
	if status != "" {
		db = db.Where("status = ?", status)
	}
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminRetryEmail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	db := database.GetDB()
	if err := db.Model(&models.EmailQueue{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "pending",
	}).Error; err != nil {
		utils.InternalError(c, "重试邮件失败")
		return
	}
	utils.SuccessMessage(c, "已重新加入队列")
}

func AdminDeleteEmail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的邮件ID")
		return
	}
	if err := database.GetDB().Delete(&models.EmailQueue{}, id).Error; err != nil {
		utils.InternalError(c, "删除失败")
		return
	}
	utils.SuccessMessage(c, "邮件记录已删除")
}

func AdminGetSettings(c *gin.Context) {
	var settings []models.SystemConfig
	database.GetDB().Where("category = ? OR category IS NULL", "").Find(&settings)
	result := make(map[string]string)
	for _, s := range settings {
		if sensitiveSettingKeys[s.Key] && s.Value != "" {
			result[s.Key] = "****"
		} else {
			result[s.Key] = s.Value
		}
	}
	utils.Success(c, result)
}

var sensitiveSettingKeys = map[string]bool{
	"smtp_password":             true,
	"pay_alipay_private_key":    true,
	"pay_alipay_public_key":     true,
	"pay_wechat_api_key":        true,
	"pay_epay_secret_key":       true,
	"pay_stripe_secret_key":     true,
	"pay_stripe_webhook_secret": true,
	"backup_github_token":       true,
	"notify_telegram_bot_token": true,
}

func AdminUpdateSettings(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	db := database.GetDB()
	for k, v := range req {
		strVal := fmt.Sprintf("%v", v)
		// 如果是敏感字段且前端传回掩码或为空，则跳过不更新
		if sensitiveSettingKeys[k] && (strVal == "" || strings.Contains(strVal, "****")) {
			continue
		}
		// 使用 Updates(map) 以避免触发全字段覆盖，仅更新 value 字段
		// 这样可以保留原有的 category, display_name 等信息
		result := db.Model(&models.SystemConfig{}).Where("`key` = ?", k).Updates(map[string]interface{}{"value": strVal})
		if result.Error == nil && result.RowsAffected == 0 {
			// 如果记录不存在，则创建新记录，默认 category 为空
			db.Create(&models.SystemConfig{Key: k, Value: strVal, Category: ""})
		}
	}
	utils.CreateAuditLog(c, "update_settings", "settings", 0, "更新系统设置")
	utils.InvalidateSettingsCache()
	utils.SuccessMessage(c, "设置已更新")
}

func AdminListAnnouncements(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.Announcement
	var total int64
	db := database.GetDB().Model(&models.Announcement{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminCreateAnnouncement(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Type    string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	ann := models.Announcement{
		Title:    req.Title,
		Content:  req.Content,
		Type:     req.Type,
		IsActive: true,
	}
	if err := database.GetDB().Create(&ann).Error; err != nil {
		utils.InternalError(c, "创建公告失败")
		return
	}
	utils.Success(c, ann)
}

func AdminUpdateAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的公告ID")
		return
	}
	var ann models.Announcement
	if err := database.GetDB().First(&ann, id).Error; err != nil {
		utils.NotFound(c, "公告不存在")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	allowed := map[string]bool{
		"title": true, "content": true, "type": true, "is_active": true, "sort_order": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	if err := database.GetDB().Model(&ann).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新公告失败")
		return
	}
	utils.Success(c, ann)
}

func AdminDeleteAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	if err := database.GetDB().Delete(&models.Announcement{}, id).Error; err != nil {
		utils.InternalError(c, "删除公告失败")
		return
	}
	utils.SuccessMessage(c, "公告已删除")
}

func AdminRevenueStats(c *gin.Context) {
	db := database.GetDB()
	var totalRevenue float64
	db.Model(&models.Order{}).Where("status IN ?", []string{"paid", "completed"}).Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&totalRevenue)
	var todayRevenue float64
	today := time.Now().Format("2006-01-02")
	db.Model(&models.Order{}).Where("status IN ? AND DATE(payment_time) = ?", []string{"paid", "completed"}, today).
		Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&todayRevenue)
	var monthRevenue float64
	monthStart := time.Now().Format("2006-01") + "-01"
	db.Model(&models.Order{}).Where("status IN ? AND payment_time >= ?", []string{"paid", "completed"}, monthStart).
		Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&monthRevenue)
	var orderCount int64
	db.Model(&models.Order{}).Where("status IN ?", []string{"paid", "completed"}).Count(&orderCount)
	utils.Success(c, gin.H{
		"total_revenue":     totalRevenue,
		"today_revenue":     todayRevenue,
		"monthly_revenue":   monthRevenue,
		"paid_orders_count": orderCount,
	})
}

func AdminUserStats(c *gin.Context) {
	db := database.GetDB()
	var totalUsers int64
	db.Model(&models.User{}).Count(&totalUsers)
	var activeUsers int64
	db.Model(&models.User{}).Where("is_active = ?", true).Count(&activeUsers)
	var todayNew int64
	today := time.Now().Format("2006-01-02")
	db.Model(&models.User{}).Where("DATE(created_at) = ?", today).Count(&todayNew)
	var paidUsers int64
	db.Model(&models.Order{}).Where("status = ?", "paid").Distinct("user_id").Count(&paidUsers)
	utils.Success(c, gin.H{
		"total_users":     totalUsers,
		"active_users":    activeUsers,
		"today_new_users": todayNew,
		"paid_users":      paidUsers,
	})
}

func AdminAuditLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.AuditLog
	var total int64
	db := database.GetDB().Model(&models.AuditLog{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminLoginLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.LoginHistory
	var total int64
	db := database.GetDB().Model(&models.LoginHistory{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminRegistrationLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.RegistrationLog
	var total int64
	db := database.GetDB().Model(&models.RegistrationLog{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminSubscriptionLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.SubscriptionLog
	var total int64
	db := database.GetDB().Model(&models.SubscriptionLog{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminBalanceLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.BalanceLog
	var total int64
	db := database.GetDB().Model(&models.BalanceLog{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminCommissionLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.CommissionLog
	var total int64
	db := database.GetDB().Model(&models.CommissionLog{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminSystemLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.SystemLog
	var total int64
	db := database.GetDB().Model(&models.SystemLog{})
	if level := c.Query("level"); level != "" {
		db = db.Where("level = ?", level)
	}
	if module := c.Query("module"); module != "" {
		db = db.Where("module = ?", module)
	}
	db.Count(&total)
	db.Order("created_at DESC").Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminMonitoring(c *gin.Context) {
	db := database.GetDB()
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	var nodeCount int64
	db.Model(&models.Node{}).Count(&nodeCount)
	var activeSubCount int64
	db.Model(&models.Subscription{}).Where("is_active = ? AND expire_time > ?", true, time.Now()).Count(&activeSubCount)
	var pendingTickets int64
	db.Model(&models.Ticket{}).Where("status = ?", "pending").Count(&pendingTickets)
	var pendingOrders int64
	db.Model(&models.Order{}).Where("status = ?", "pending").Count(&pendingOrders)
	utils.Success(c, gin.H{
		"user_count":           userCount,
		"node_count":           nodeCount,
		"active_subscriptions": activeSubCount,
		"pending_tickets":      pendingTickets,
		"pending_orders":       pendingOrders,
	})
}

func AdminCreateBackup(c *gin.Context) {
	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		utils.InternalError(c, "创建备份目录失败: "+err.Error())
		return
	}

	// Find the SQLite database file
	srcPath := "cboard.db"
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		utils.InternalError(c, "数据库文件不存在，仅支持 SQLite 备份")
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	dbBackupPath := filepath.Join(backupDir, fmt.Sprintf("cboard_backup_%s.db", timestamp))

	// Copy database file
	src, err := os.Open(srcPath)
	if err != nil {
		utils.InternalError(c, "打开数据库失败: "+err.Error())
		return
	}
	defer src.Close()

	// #nosec G304 -- dbBackupPath is server-generated under fixed backups directory.
	dst, err := os.Create(dbBackupPath)
	if err != nil {
		utils.InternalError(c, "创建备份文件失败: "+err.Error())
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		utils.InternalError(c, "备份失败: "+err.Error())
		return
	}

	// Create ZIP file containing db + .env
	zipPath := filepath.Join(backupDir, fmt.Sprintf("cboard_backup_%s.zip", timestamp))
	// #nosec G304 -- zipPath is server-generated under fixed backups directory.
	zipFile, err := os.Create(zipPath)
	if err != nil {
		utils.InternalError(c, "创建ZIP文件失败: "+err.Error())
		return
	}
	zipWriter := zip.NewWriter(zipFile)

	// Add database to ZIP
	if err := addFileToZip(zipWriter, dbBackupPath, filepath.Base(dbBackupPath)); err != nil {
		utils.InternalError(c, "添加数据库到ZIP失败: "+err.Error())
		return
	}

	// 注意：不备份 .env 文件以防止敏感信息泄露（数据库密码、API密钥、支付密钥等）
	// 如需备份配置，请使用加密存储或单独的安全备份方案

	if err := zipWriter.Close(); err != nil {
		utils.InternalError(c, "关闭ZIP写入失败: "+err.Error())
		return
	}
	if err := zipFile.Close(); err != nil {
		utils.InternalError(c, "关闭ZIP文件失败: "+err.Error())
		return
	}

	info, _ := os.Stat(dbBackupPath)
	zipInfo, _ := os.Stat(zipPath)

	response := gin.H{
		"filename":   filepath.Base(dbBackupPath),
		"size":       info.Size(),
		"created_at": time.Now(),
	}

	// Check if GitHub backup is enabled
	settings := utils.GetSettings("backup_github_enabled", "backup_github_token", "backup_github_repo")
	if settings["backup_github_enabled"] == "true" || settings["backup_github_enabled"] == "1" {
		token := settings["backup_github_token"]
		repo := settings["backup_github_repo"]

		if token != "" && repo != "" {
			// Parse owner/repo
			parts := strings.SplitN(repo, "/", 2)
			if len(parts) == 2 {
				owner := parts[0]
				repoName := parts[1]

				// Generate task ID
				taskID := uuid.New().String()

				// Create upload status
				statusManager := git.GetUploadStatusManager()
				status := &git.UploadStatus{
					Status:    "uploading",
					Progress:  0,
					Message:   "准备上传...",
					StartTime: time.Now(),
					FileName:  filepath.Base(zipPath),
					FileSize:  zipInfo.Size(),
				}
				statusManager.SetStatus(taskID, status)

				// Start async upload
				go func() {
					client := git.NewClient(git.PlatformGitHub, token, owner, repoName)
					err := client.UploadBackupWithProgress(zipPath, func(progress int, message string) {
						statusManager.UpdateStatus(taskID, "uploading", message, progress)
					})

					if err != nil {
						statusManager.UpdateError(taskID, err)
					} else {
						statusManager.UpdateStatus(taskID, "success", "上传成功", 100)
					}
				}()

				response["task_id"] = taskID
				response["github_upload"] = "started"
			}
		}
	}

	utils.Success(c, response)
}

// addFileToZip adds a file to the zip archive
func addFileToZip(zipWriter *zip.Writer, filePath, nameInZip string) error {
	if strings.Contains(nameInZip, "..") || strings.ContainsAny(nameInZip, `/\`) {
		return fmt.Errorf("invalid zip entry name")
	}
	// #nosec G304 -- filePath comes from controlled backup generation flow.
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := zipWriter.Create(nameInZip)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

func AdminListBackups(c *gin.Context) {
	backupDir := "backups"
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		utils.Success(c, []interface{}{})
		return
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		utils.Success(c, []interface{}{})
		return
	}

	type BackupInfo struct {
		Filename  string    `json:"filename"`
		Size      int64     `json:"size"`
		CreatedAt time.Time `json:"created_at"`
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Filename:  entry.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	// Sort by newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	utils.Success(c, backups)
}

func AdminGetUploadStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.BadRequest(c, "任务ID不能为空")
		return
	}
	statusManager := git.GetUploadStatusManager()
	status, exists := statusManager.GetStatus(taskID)
	if !exists {
		utils.NotFound(c, "未找到该上传任务")
		return
	}
	utils.Success(c, status)
}

func AdminTestGitHubConnection(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
		Repo  string `json:"repo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Fall back to saved settings
		settings := utils.GetSettings("backup_github_token", "backup_github_repo")
		req.Token = settings["backup_github_token"]
		req.Repo = settings["backup_github_repo"]
	}
	if req.Token == "" {
		utils.BadRequest(c, "Token不能为空")
		return
	}
	if req.Repo == "" {
		utils.BadRequest(c, "仓库地址不能为空")
		return
	}
	repo := strings.TrimSpace(req.Repo)
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "http://github.com/")
	repo = strings.TrimSuffix(repo, ".git")
	repo = strings.Trim(repo, "/")
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		utils.BadRequest(c, "仓库地址格式错误，应为 owner/repo")
		return
	}
	client := git.NewClient(git.PlatformGitHub, req.Token, parts[0], parts[1])
	if err := client.TestConnection(); err != nil {
		utils.BadRequest(c, "GitHub 连接测试失败: "+err.Error())
		return
	}
	utils.Success(c, gin.H{"message": "GitHub 连接测试成功"})
}

func AdminBackfillLocations(c *gin.Context) {
	db := database.GetDB()
	backfilled := map[string]int64{}

	updateTable := func(tableName string) error {
		rows, err := db.Table(tableName).Select("id, ip_address").Where("(location IS NULL OR location = '') AND ip_address IS NOT NULL AND ip_address != ''").Rows()
		if err != nil {
			return err
		}
		defer rows.Close()

		var count int64
		for rows.Next() {
			var id uint
			var ip string
			if err := rows.Scan(&id, &ip); err != nil {
				continue
			}
			location := utils.GetIPLocation(ip)
			if location == "" {
				continue
			}
			if err := db.Table(tableName).Where("id = ?", id).Update("location", location).Error; err == nil {
				count++
			}
		}
		backfilled[tableName] = count
		return nil
	}

	for _, tableName := range []string{"login_history", "user_activities", "registration_logs", "subscription_logs", "balance_logs"} {
		if err := updateTable(tableName); err != nil {
			utils.InternalError(c, "回填 "+tableName+" 失败: "+err.Error())
			return
		}
	}

	utils.Success(c, gin.H{
		"backfilled": backfilled,
		"message":    "历史地区数据回填完成",
	})
}

// ==================== Create User ====================

func AdminUpdateGeoIP(c *gin.Context) {
	resources := map[string]string{
		"geoip.dat":    "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.dat",
		"geosite.dat":  "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat",
		"geoip.metadb": "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb",
	}

	if err := os.MkdirAll(filepath.Join("uploads", "config"), 0750); err != nil {
		utils.InternalError(c, "创建 GeoIP 目录失败: "+err.Error())
		return
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}
	updated := make([]string, 0, len(resources))
	for fileName, fileURL := range resources {
		resp, err := httpClient.Get(fileURL)
		if err != nil {
			utils.InternalError(c, "下载 "+fileName+" 失败: "+err.Error())
			return
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			utils.InternalError(c, "下载 "+fileName+" 失败: "+resp.Status)
			return
		}
		targetPath := filepath.Join("uploads", "config", fileName)
		file, err := os.Create(targetPath)
		if err != nil {
			resp.Body.Close()
			utils.InternalError(c, "写入 "+fileName+" 失败: "+err.Error())
			return
		}
		if _, err := io.Copy(file, resp.Body); err != nil {
			file.Close()
			resp.Body.Close()
			utils.InternalError(c, "保存 "+fileName+" 失败: "+err.Error())
			return
		}
		file.Close()
		resp.Body.Close()
		updated = append(updated, fileName)
	}

	utils.Success(c, gin.H{
		"updated": updated,
		"message": "GeoIP 数据更新成功",
	})
}

func AdminCreateUser(c *gin.Context) {
	var req struct {
		Username    string     `json:"username" binding:"required,min=3,max=50"`
		Email       string     `json:"email" binding:"required,email"`
		Password    string     `json:"password" binding:"required,min=6"`
		Balance     float64    `json:"balance"`
		IsAdmin     bool       `json:"is_admin"`
		IsActive    bool       `json:"is_active"`
		Notes       string     `json:"notes"`
		ExpireTime  *time.Time `json:"expire_time"`
		DeviceLimit int        `json:"device_limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	db := database.GetDB()
	var count int64
	db.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		utils.Conflict(c, "邮箱已存在")
		return
	}
	db.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		utils.Conflict(c, "用户名已存在")
		return
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.InternalError(c, "密码加密失败")
		return
	}

	user := models.User{
		Username:                    req.Username,
		Email:                       req.Email,
		Password:                    hashed,
		Balance:                     req.Balance,
		IsAdmin:                     req.IsAdmin,
		IsActive:                    req.IsActive,
		IsVerified:                  true,
		Theme:                       "light",
		Language:                    "zh-CN",
		Timezone:                    "Asia/Shanghai",
		SpecialNodeSubscriptionType: "both",
	}
	if req.Notes != "" {
		user.Notes = &req.Notes
	}

	if err := db.Create(&user).Error; err != nil {
		utils.InternalError(c, "创建用户失败")
		return
	}

	// Auto-create subscription for new user
	subURL := utils.GenerateRandomString(64)

	// 设置到期时间和设备限制
	expireTime := time.Now()
	if req.ExpireTime != nil {
		expireTime = *req.ExpireTime
	}

	deviceLimit := 3
	if req.DeviceLimit > 0 {
		deviceLimit = req.DeviceLimit
	}

	subscription := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: subURL,
		DeviceLimit:     deviceLimit,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      expireTime,
	}
	if err := db.Create(&subscription).Error; err != nil {
		utils.InternalError(c, "创建订阅失败")
		return
	}

	// 发送账户创建通知邮件
	go services.NotifyUserDirect(user.Email, "admin_create_user", map[string]string{
		"username": user.Username, "email": user.Email,
	})
	go services.NotifyAdmin("admin_create_user", map[string]string{
		"username": user.Username, "email": user.Email,
	})

	utils.CreateAuditLog(c, "create_user", "user", user.ID, fmt.Sprintf("管理员创建用户: %s (%s)", user.Username, user.Email))
	utils.Success(c, user)
}

// ==================== Reset User Password ====================

func AdminResetUserPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	var req struct {
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.InternalError(c, "密码加密失败")
		return
	}
	if err := db.Model(&user).Update("password", hashed).Error; err != nil {
		utils.InternalError(c, "重置密码失败")
		return
	}
	utils.CreateAuditLog(c, "reset_password", "user", uint(id), fmt.Sprintf("重置用户密码: %s", user.Username))
	utils.SuccessMessage(c, "密码已重置")
}

// ==================== Test Email ====================

func AdminSendTestEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	subject, body := services.RenderEmail("test", map[string]string{})
	err := services.SendEmail(req.Email, subject, body)
	if err != nil {
		utils.InternalError(c, "发送失败: "+err.Error())
		return
	}
	utils.SuccessMessage(c, "测试邮件已发送至 "+req.Email)
}

func AdminTestTelegram(c *gin.Context) {
	if err := services.SendTestTelegram(); err != nil {
		utils.InternalError(c, "发送失败: "+err.Error())
		return
	}
	utils.SuccessMessage(c, "Telegram 测试消息已发送")
}

// ==================== Update Subscription ====================

func AdminUpdateSubscription(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订阅ID")
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		utils.NotFound(c, "订阅不存在")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	allowed := map[string]bool{
		"device_limit": true, "is_active": true, "expire_time": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			// 处理 expire_time 的时间格式转换
			if k == "expire_time" {
				if expireTimeStr, ok := v.(string); ok && expireTimeStr != "" {
					if expireTime, err := time.Parse(time.RFC3339, expireTimeStr); err == nil {
						updates[k] = expireTime
					}
				}
			} else {
				updates[k] = v
			}
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "没有可更新的字段")
		return
	}

	// 如果更新了 expire_time，需要同步更新 status
	if expireTime, ok := updates["expire_time"].(time.Time); ok {
		now := time.Now()
		if expireTime.After(now) {
			// 未过期
			if time.Until(expireTime) <= 7*24*time.Hour {
				updates["status"] = "expiring"
			} else {
				updates["status"] = "active"
			}
			// 确保 is_active 为 true
			if _, hasIsActive := updates["is_active"]; !hasIsActive {
				updates["is_active"] = true
			}
		} else {
			// 已过期
			updates["status"] = "expired"
		}
	}

	if err := db.Model(&sub).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新订阅失败")
		return
	}
	adminID := c.GetUint("user_id")
	utils.CreateSubscriptionLog(sub.ID, sub.UserID, "update", "admin", &adminID, "管理员更新订阅设置", nil, nil)
	utils.CreateAuditLog(c, "update_subscription", "subscription", uint(id), fmt.Sprintf("更新订阅 (用户ID: %d)", sub.UserID))
	utils.Success(c, sub)
}

// ==================== Admin Send Subscription Email ====================

func AdminSendSubscriptionEmail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订阅ID")
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		utils.NotFound(c, "订阅不存在")
		return
	}
	var user models.User
	if err := db.First(&user, sub.UserID).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}
	baseURL := getSubscriptionBaseURL()
	if baseURL == "" {
		utils.BadRequest(c, "系统未配置域名")
		return
	}
	universalURL := fmt.Sprintf("%s/api/v1/sub/%s", baseURL, sub.SubscriptionURL)
	clashURL := fmt.Sprintf("%s/api/v1/sub/clash/%s", baseURL, sub.SubscriptionURL)
	subject, body := services.RenderEmail("subscription", map[string]string{
		"clash_url": clashURL, "universal_url": universalURL,
		"expire_time": sub.ExpireTime.Format("2006-01-02 15:04"),
	})
	go services.QueueEmail(user.Email, subject, body, "subscription")
	utils.SuccessMessage(c, "订阅信息已发送至 "+user.Email)
}

// ==================== Admin Delete User (Full) ====================

func AdminDeleteUserFull(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	db := database.GetDB()
	var user models.User
	userExists := db.First(&user, id).Error == nil

	tx := db.Begin()
	if tx.Error != nil {
		utils.InternalError(c, "创建事务失败")
		return
	}
	rollbackWithErr := func(err error) bool {
		if err == nil {
			return false
		}
		tx.Rollback()
		utils.InternalError(c, "删除用户失败")
		return true
	}

	// Delete ticket replies (via user's tickets)
	var ticketIDs []uint
	if rollbackWithErr(tx.Model(&models.Ticket{}).Where("user_id = ?", id).Pluck("id", &ticketIDs).Error) {
		return
	}
	if len(ticketIDs) > 0 {
		if rollbackWithErr(tx.Where("ticket_id IN ?", ticketIDs).Delete(&models.TicketReply{}).Error) {
			return
		}
	}

	// Delete all related data by user_id
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.PaymentTransaction{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.Notification{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.UserActivity{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.InviteCode{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("inviter_id = ? OR invitee_id = ?", id, id).Delete(&models.InviteRelation{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("inviter_id = ? OR invitee_id = ?", id, id).Delete(&models.CommissionLog{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.RegistrationLog{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.SubscriptionLog{}).Error) {
		return
	}

	if userExists {
		if rollbackWithErr(tx.Where("username = ? OR username = ?", user.Email, user.Username).Delete(&models.LoginAttempt{}).Error) {
			return
		}
		if rollbackWithErr(tx.Where("email = ?", user.Email).Delete(&models.VerificationCode{}).Error) {
			return
		}
	}

	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.Order{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.Device{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.SubscriptionReset{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.Subscription{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.Ticket{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.BalanceLog{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.LoginHistory{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.RechargeRecord{}).Error) {
		return
	}
	if rollbackWithErr(tx.Where("user_id = ?", id).Delete(&models.UserCustomNode{}).Error) {
		return
	}

	if userExists {
		if rollbackWithErr(tx.Delete(&user).Error) {
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		utils.InternalError(c, "删除用户失败")
		return
	}

	if userExists {
		go services.NotifyUserDirect(user.Email, "account_deleted", nil)
		utils.CreateAuditLog(c, "delete_user_full", "user", uint(id),
			fmt.Sprintf("完全删除用户: %s (%s)", user.Username, user.Email))
	} else {
		utils.CreateAuditLog(c, "delete_user_full", "user", uint(id),
			fmt.Sprintf("清理孤立数据: 用户ID %d", id))
	}
	utils.SuccessMessage(c, "用户及所有关联数据已删除")
}

// ==================== Admin Set Subscription Expire Time ====================

func AdminSetSubscriptionExpireTime(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订阅ID")
		return
	}
	var req struct {
		ExpireTime string `json:"expire_time" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	expireTime, err := time.Parse("2006-01-02T15:04:05Z", req.ExpireTime)
	if err != nil {
		expireTime, err = time.Parse("2006-01-02 15:04:05", req.ExpireTime)
		if err != nil {
			expireTime, err = time.Parse("2006-01-02", req.ExpireTime)
			if err != nil {
				utils.BadRequest(c, "时间格式错误，支持: 2006-01-02 或 2006-01-02 15:04:05")
				return
			}
		}
	}

	// Validate date range
	maxFuture := time.Now().AddDate(10, 0, 0)
	if expireTime.After(maxFuture) {
		utils.BadRequest(c, "到期时间不能超过 10 年")
		return
	}

	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		utils.NotFound(c, "订阅不存在")
		return
	}

	updates := map[string]interface{}{"expire_time": expireTime}
	if expireTime.After(time.Now()) {
		updates["is_active"] = true
		updates["status"] = "active"
	}
	if err := db.Model(&sub).Updates(updates).Error; err != nil {
		utils.InternalError(c, "设置订阅到期时间失败")
		return
	}
	adminID := c.GetUint("user_id")
	utils.CreateSubscriptionLog(sub.ID, sub.UserID, "update", "admin", &adminID, fmt.Sprintf("管理员设置到期时间: %s", expireTime.Format("2006-01-02")), nil, nil)
	utils.CreateAuditLog(c, "set_expire_time", "subscription", uint(id), fmt.Sprintf("设置订阅到期时间: %s (用户ID: %d)", expireTime.Format("2006-01-02"), sub.UserID))
	utils.Success(c, gin.H{"expire_time": expireTime})
}

// ==================== Public Announcements ====================

func ListPublicAnnouncements(c *gin.Context) {
	db := database.GetDB()
	var items []models.Announcement
	db.Where("is_active = ?", true).Order("created_at DESC").Limit(10).Find(&items)
	utils.Success(c, items)
}

// ==================== Financial Report ====================

func AdminFinancialReport(c *gin.Context) {
	db := database.GetDB()

	period := c.DefaultQuery("period", "month")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Default date range
	now := time.Now()
	var start, end time.Time
	if startDate != "" {
		s, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			utils.BadRequest(c, "start_date 格式错误，应为 YYYY-MM-DD")
			return
		}
		start = s
	} else {
		start = now.AddDate(0, 0, -29)
	}
	if endDate != "" {
		e, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			utils.BadRequest(c, "end_date 格式错误，应为 YYYY-MM-DD")
			return
		}
		end = e.Add(24*time.Hour - time.Second)
	} else {
		end = now
	}
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	// ---- Summary ----
	var totalRevenue float64
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ? AND DATE(payment_time) <= ?", "paid", startStr, endStr).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)

	var totalOrders int64
	db.Model(&models.Order{}).
		Where("DATE(created_at) >= ? AND DATE(created_at) <= ?", startStr, endStr).
		Count(&totalOrders)

	var paidOrders int64
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ? AND DATE(payment_time) <= ?", "paid", startStr, endStr).
		Count(&paidOrders)
	var refundedOrders int64
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(updated_at) >= ? AND DATE(updated_at) <= ?", "refunded", startStr, endStr).
		Count(&refundedOrders)

	var avgOrderAmount float64
	if paidOrders > 0 {
		avgOrderAmount = totalRevenue / float64(paidOrders)
	}

	var totalRecharge float64
	db.Model(&models.RechargeRecord{}).
		Where("status = ? AND DATE(paid_at) >= ? AND DATE(paid_at) <= ?", "paid", startStr, endStr).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRecharge)

	var totalRechargeCount int64
	db.Model(&models.RechargeRecord{}).
		Where("status = ? AND DATE(paid_at) >= ? AND DATE(paid_at) <= ?", "paid", startStr, endStr).
		Count(&totalRechargeCount)

	var newUsers int64
	db.Model(&models.User{}).
		Where("DATE(created_at) >= ? AND DATE(created_at) <= ?", startStr, endStr).
		Count(&newUsers)

	var newSubscriptions int64
	db.Model(&models.Subscription{}).
		Where("DATE(created_at) >= ? AND DATE(created_at) <= ?", startStr, endStr).
		Count(&newSubscriptions)

	summary := gin.H{
		"total_revenue":        totalRevenue,
		"total_orders":         totalOrders,
		"paid_orders":          paidOrders,
		"refunded_orders":      refundedOrders,
		"average_order_amount": avgOrderAmount,
		"total_recharge":       totalRecharge,
		"total_recharge_count": totalRechargeCount,
		"new_users":            newUsers,
		"new_subscriptions":    newSubscriptions,
	}

	// ---- Revenue Chart ----
	var dateExpr string
	switch period {
	case "day":
		dateExpr = "DATE(payment_time)"
	case "week":
		dateExpr = "DATE(payment_time, 'weekday 0', '-6 days')"
	default:
		dateExpr = "strftime('%Y-%m', payment_time)"
	}

	type ChartPoint struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue"`
		Orders  int64   `json:"orders"`
	}
	var revenueChart []ChartPoint
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ? AND DATE(payment_time) <= ?", "paid", startStr, endStr).
		Select(dateExpr + " as date, COALESCE(SUM(amount), 0) as revenue, COUNT(*) as orders").
		Group(dateExpr).
		Order("date ASC").
		Scan(&revenueChart)
	// Recharge per period for chart
	type RechargePoint struct {
		Date     string  `json:"date"`
		Recharge float64 `json:"recharge"`
	}
	var rechargeByDate []RechargePoint
	var rechargeDateExpr string
	switch period {
	case "day":
		rechargeDateExpr = "DATE(paid_at)"
	case "week":
		rechargeDateExpr = "DATE(paid_at, 'weekday 0', '-6 days')"
	default:
		rechargeDateExpr = "strftime('%Y-%m', paid_at)"
	}
	db.Model(&models.RechargeRecord{}).
		Where("status = ? AND DATE(paid_at) >= ? AND DATE(paid_at) <= ?", "paid", startStr, endStr).
		Select(rechargeDateExpr + " as date, COALESCE(SUM(amount), 0) as recharge").
		Group(rechargeDateExpr).
		Order("date ASC").
		Scan(&rechargeByDate)

	rechargeMap := make(map[string]float64)
	for _, r := range rechargeByDate {
		rechargeMap[r.Date] = r.Recharge
	}
	type ChartPointFull struct {
		Date     string  `json:"date"`
		Revenue  float64 `json:"revenue"`
		Orders   int64   `json:"orders"`
		Recharge float64 `json:"recharge"`
	}
	chartFull := make([]ChartPointFull, 0, len(revenueChart))
	for _, cp := range revenueChart {
		chartFull = append(chartFull, ChartPointFull{
			Date:     cp.Date,
			Revenue:  cp.Revenue,
			Orders:   cp.Orders,
			Recharge: rechargeMap[cp.Date],
		})
	}

	// ---- Payment Method Stats ----
	type PaymentMethodStat struct {
		Method string  `json:"method"`
		Count  int64   `json:"count"`
		Amount float64 `json:"amount"`
	}
	var paymentMethodStats []PaymentMethodStat
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ? AND DATE(payment_time) <= ? AND payment_method_name IS NOT NULL", "paid", startStr, endStr).
		Select("COALESCE(payment_method_name, '未知') as method, COUNT(*) as count, COALESCE(SUM(amount), 0) as amount").
		Group("payment_method_name").
		Order("amount DESC").
		Scan(&paymentMethodStats)
	// ---- Package Stats ----
	type PackageStat struct {
		PackageName string  `json:"package_name"`
		Count       int64   `json:"count"`
		Amount      float64 `json:"amount"`
	}
	var packageStats []PackageStat
	db.Model(&models.Order{}).
		Joins("LEFT JOIN packages ON packages.id = orders.package_id").
		Where("orders.status = ? AND DATE(orders.payment_time) >= ? AND DATE(orders.payment_time) <= ?", "paid", startStr, endStr).
		Select("COALESCE(packages.name, '未知套餐') as package_name, COUNT(*) as count, COALESCE(SUM(orders.amount), 0) as amount").
		Group("orders.package_id").
		Order("amount DESC").
		Scan(&packageStats)

	// ---- Top Users ----
	type TopUser struct {
		UserID     uint    `json:"user_id"`
		Username   string  `json:"username"`
		TotalSpent float64 `json:"total_spent"`
		OrderCount int64   `json:"order_count"`
	}
	var topUsers []TopUser
	db.Model(&models.Order{}).
		Joins("LEFT JOIN users ON users.id = orders.user_id").
		Where("orders.status = ? AND DATE(orders.payment_time) >= ? AND DATE(orders.payment_time) <= ?", "paid", startStr, endStr).
		Select("orders.user_id, COALESCE(users.username, '未知') as username, COALESCE(SUM(orders.amount), 0) as total_spent, COUNT(*) as order_count").
		Group("orders.user_id").
		Order("total_spent DESC").
		Limit(10).
		Scan(&topUsers)

	utils.Success(c, gin.H{
		"summary":              summary,
		"revenue_chart":        chartFull,
		"payment_method_stats": paymentMethodStats,
		"package_stats":        packageStats,
		"top_users":            topUsers,
	})
}

func AdminExportFinancialReport(c *gin.Context) {
	db := database.GetDB()

	period := c.DefaultQuery("period", "month")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	now := time.Now()
	var start, end time.Time
	if startDate != "" {
		s, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			utils.BadRequest(c, "start_date 格式错误")
			return
		}
		start = s
	} else {
		start = now.AddDate(0, 0, -29)
	}
	if endDate != "" {
		e, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			utils.BadRequest(c, "end_date 格式错误")
			return
		}
		end = e.Add(24*time.Hour - time.Second)
	} else {
		end = now
	}
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	var dateExpr string
	switch period {
	case "day":
		dateExpr = "DATE(payment_time)"
	case "week":
		dateExpr = "DATE(payment_time, 'weekday 0', '-6 days')"
	default:
		dateExpr = "strftime('%Y-%m', payment_time)"
	}

	type Row struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue"`
		Orders  int64   `json:"orders"`
	}
	var rows []Row
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ? AND DATE(payment_time) <= ?", "paid", startStr, endStr).
		Select(dateExpr + " as date, COALESCE(SUM(amount), 0) as revenue, COUNT(*) as orders").
		Group(dateExpr).
		Order("date ASC").
		Scan(&rows)

	// Recharge per period
	var rechargeDateExpr string
	switch period {
	case "day":
		rechargeDateExpr = "DATE(paid_at)"
	case "week":
		rechargeDateExpr = "DATE(paid_at, 'weekday 0', '-6 days')"
	default:
		rechargeDateExpr = "strftime('%Y-%m', paid_at)"
	}
	type RRow struct {
		Date     string  `json:"date"`
		Recharge float64 `json:"recharge"`
	}
	var rrows []RRow
	db.Model(&models.RechargeRecord{}).
		Where("status = ? AND DATE(paid_at) >= ? AND DATE(paid_at) <= ?", "paid", startStr, endStr).
		Select(rechargeDateExpr + " as date, COALESCE(SUM(amount), 0) as recharge").
		Group(rechargeDateExpr).
		Order("date ASC").
		Scan(&rrows)
	rechargeMap := make(map[string]float64)
	for _, r := range rrows {
		rechargeMap[r.Date] = r.Recharge
	}

	// New users per period
	var userDateExpr string
	switch period {
	case "day":
		userDateExpr = "DATE(created_at)"
	case "week":
		userDateExpr = "DATE(created_at, 'weekday 0', '-6 days')"
	default:
		userDateExpr = "strftime('%Y-%m', created_at)"
	}
	type URow struct {
		Date     string `json:"date"`
		NewUsers int64  `json:"new_users"`
	}
	var urows []URow
	db.Model(&models.User{}).
		Where("DATE(created_at) >= ? AND DATE(created_at) <= ?", startStr, endStr).
		Select(userDateExpr + " as date, COUNT(*) as new_users").
		Group(userDateExpr).
		Order("date ASC").
		Scan(&urows)
	userMap := make(map[string]int64)
	for _, u := range urows {
		userMap[u.Date] = u.NewUsers
	}

	filename := fmt.Sprintf("financial_report_%s.csv", now.Format("2006-01-02"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	// BOM for Excel UTF-8
	if _, err := c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		utils.InternalError(c, "导出失败")
		return
	}
	if _, err := c.Writer.WriteString("日期,收入,订单数,充值,新用户\n"); err != nil {
		utils.InternalError(c, "导出失败")
		return
	}
	for _, row := range rows {
		line := fmt.Sprintf("%s,%.2f,%d,%.2f,%d\n",
			sanitizeCSVCell(row.Date), row.Revenue, row.Orders, rechargeMap[row.Date], userMap[row.Date])
		if _, err := c.Writer.WriteString(line); err != nil {
			utils.InternalError(c, "导出失败")
			return
		}
	}
}

// ==================== Region Statistics ====================

func AdminRegionStats(c *gin.Context) {
	db := database.GetDB()

	type RegionCount struct {
		Location string `json:"location"`
		Count    int64  `json:"count"`
	}

	var regions []RegionCount
	db.Model(&models.LoginHistory{}).
		Select("COALESCE(location, '未知') as location, COUNT(DISTINCT user_id) as count").
		Where("location IS NOT NULL AND location != ''").
		Group("location").
		Order("count DESC").
		Limit(20).
		Find(&regions)

	utils.Success(c, regions)
}

// ==================== Batch Operations ====================

func AdminBatchUserAction(c *gin.Context) {
	var req struct {
		UserIDs []uint                 `json:"user_ids" binding:"required"`
		Action  string                 `json:"action" binding:"required"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if len(req.UserIDs) == 0 {
		utils.BadRequest(c, "请选择用户")
		return
	}

	db := database.GetDB()
	var affected int64

	switch req.Action {
	case "enable":
		result := db.Model(&models.User{}).Where("id IN ?", req.UserIDs).Update("is_active", true)
		affected = result.RowsAffected
		// Sync subscription status
		for _, uid := range req.UserIDs {
			var sub models.Subscription
			if db.Where("user_id = ?", uid).First(&sub).Error == nil {
				updates := map[string]interface{}{"is_active": true}
				if sub.ExpireTime.After(time.Now()) {
					updates["status"] = "active"
				} else {
					updates["status"] = "expired"
				}
				if err := db.Model(&sub).Updates(updates).Error; err != nil {
					utils.InternalError(c, "同步订阅状态失败")
					return
				}
			}
		}
	case "disable":
		result := db.Model(&models.User{}).Where("id IN ? AND is_admin = ?", req.UserIDs, false).Update("is_active", false)
		affected = result.RowsAffected
		if err := db.Model(&models.Subscription{}).Where("user_id IN ?", req.UserIDs).Updates(map[string]interface{}{
			"is_active": false, "status": "disabled",
		}).Error; err != nil {
			utils.InternalError(c, "同步订阅状态失败")
			return
		}
	case "delete":
		result := db.Where("id IN ? AND is_admin = ?", req.UserIDs, false).Delete(&models.User{})
		affected = result.RowsAffected
	case "reset_password":
		password := "123456"
		if req.Data != nil {
			if p, ok := req.Data["password"].(string); ok && p != "" {
				password = p
			}
		}
		hashed, err := utils.HashPassword(password)
		if err != nil {
			utils.InternalError(c, "密码加密失败")
			return
		}
		result := db.Model(&models.User{}).Where("id IN ?", req.UserIDs).Update("password", hashed)
		affected = result.RowsAffected
	case "set_level":
		if req.Data == nil {
			utils.BadRequest(c, "缺少等级参数")
			return
		}
		levelIDRaw, ok := req.Data["level_id"]
		if !ok {
			utils.BadRequest(c, "缺少 level_id 参数")
			return
		}
		var levelID uint
		switch v := levelIDRaw.(type) {
		case float64:
			levelID = uint(v)
		case string:
			parsed, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				utils.BadRequest(c, "无效的 level_id")
				return
			}
			levelID = uint(parsed)
		default:
			utils.BadRequest(c, "无效的 level_id 类型")
			return
		}
		result := db.Model(&models.User{}).Where("id IN ?", req.UserIDs).Update("user_level_id", levelID)
		affected = result.RowsAffected
	default:
		utils.BadRequest(c, "不支持的操作: "+req.Action)
		return
	}

	utils.CreateAuditLog(c, "batch_user_action", "user", 0, fmt.Sprintf("批量操作用户: %s, 影响 %d 个用户", req.Action, affected))
	utils.Success(c, gin.H{"affected": affected, "action": req.Action})
}

// ==================== CSV Export/Import ====================

func AdminExportUsersCSV(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.User{})

	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("username LIKE ? OR email LIKE ?", like, like)
	}
	if status := c.Query("is_active"); status != "" {
		query = query.Where("is_active = ?", status == "true")
	}

	var users []models.User
	if err := query.Order("id ASC").Find(&users).Error; err != nil {
		utils.InternalError(c, "查询用户失败: "+err.Error())
		return
	}

	filename := fmt.Sprintf("users_%s.csv", time.Now().Format("2006-01-02"))
	c.Status(200)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Write BOM for Excel compatibility
	if _, err := c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		utils.InternalError(c, "导出失败")
		return
	}

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	if err := writer.Write([]string{"ID", "用户名", "邮箱", "余额", "是否激活", "注册时间", "最后登录"}); err != nil {
		utils.InternalError(c, "导出失败")
		return
	}

	for _, u := range users {
		isActive := "否"
		if u.IsActive {
			isActive = "是"
		}
		lastLogin := ""
		if u.LastLogin != nil {
			lastLogin = u.LastLogin.Format("2006-01-02 15:04:05")
		}
		if err := writer.Write([]string{
			strconv.FormatUint(uint64(u.ID), 10),
			sanitizeCSVCell(u.Username),
			sanitizeCSVCell(u.Email),
			fmt.Sprintf("%.2f", u.Balance),
			isActive,
			u.CreatedAt.Format("2006-01-02 15:04:05"),
			lastLogin,
		}); err != nil {
			utils.InternalError(c, "导出失败")
			return
		}
	}
	if err := writer.Error(); err != nil {
		utils.InternalError(c, "导出失败")
		return
	}
}

func sanitizeCSVCell(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@':
		return "'" + v
	default:
		return v
	}
}

func AdminImportUsersCSV(c *gin.Context) {
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		utils.BadRequest(c, "请上传CSV文件")
		return
	}
	defer file.Close()

	// 限制文件大小 10MB
	const maxCSVSize = 10 * 1024 * 1024
	if fileHeader.Size > maxCSVSize {
		utils.BadRequest(c, "文件过大，最大允许 10MB")
		return
	}

	reader := csv.NewReader(io.LimitReader(file, maxCSVSize))
	// Read header
	header, err := reader.Read()
	if err != nil {
		utils.BadRequest(c, "CSV文件格式错误")
		return
	}

	// Map column indices
	colMap := make(map[string]int)
	for i, h := range header {
		// Strip BOM from first column
		h = strings.TrimPrefix(h, "\xEF\xBB\xBF")
		colMap[strings.TrimSpace(h)] = i
	}

	// Validate required columns
	requiredCols := []string{"用户名", "邮箱", "密码"}
	for _, col := range requiredCols {
		if _, ok := colMap[col]; !ok {
			utils.BadRequest(c, fmt.Sprintf("CSV缺少必要列: %s", col))
			return
		}
	}

	db := database.GetDB()
	var total, imported, skipped int
	var errors []string
	const maxRows = 5000

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("第%d行: 读取错误", total+2))
			total++
			continue
		}
		total++
		if total > maxRows {
			errors = append(errors, fmt.Sprintf("超过最大行数限制 %d 行，后续行已忽略", maxRows))
			break
		}
		rowNum := total + 1 // 1-indexed, +1 for header

		if len(record) <= colMap["用户名"] || len(record) <= colMap["邮箱"] || len(record) <= colMap["密码"] {
			errors = append(errors, fmt.Sprintf("第%d行: 列数不足", rowNum))
			skipped++
			continue
		}

		username := strings.TrimSpace(record[colMap["用户名"]])
		email := strings.TrimSpace(record[colMap["邮箱"]])
		password := strings.TrimSpace(record[colMap["密码"]])

		if username == "" || email == "" {
			errors = append(errors, fmt.Sprintf("第%d行: 用户名或邮箱为空", rowNum))
			skipped++
			continue
		}

		// Validate email
		if _, err := mail.ParseAddress(email); err != nil {
			errors = append(errors, fmt.Sprintf("第%d行: 邮箱格式无效 (%s)", rowNum, email))
			skipped++
			continue
		}

		// Check duplicates
		var count int64
		db.Model(&models.User{}).Where("email = ? OR username = ?", email, username).Count(&count)
		if count > 0 {
			errors = append(errors, fmt.Sprintf("第%d行: 用户名或邮箱已存在 (%s / %s)", rowNum, username, email))
			skipped++
			continue
		}

		if password == "" {
			password = utils.GenerateRandomString(12)
		}
		hashed, err := utils.HashPassword(password)
		if err != nil {
			errors = append(errors, fmt.Sprintf("第%d行: 密码加密失败", rowNum))
			skipped++
			continue
		}

		user := models.User{
			Username:                    username,
			Email:                       email,
			Password:                    hashed,
			IsActive:                    true,
			IsVerified:                  true,
			Theme:                       "light",
			Language:                    "zh-CN",
			Timezone:                    "Asia/Shanghai",
			SpecialNodeSubscriptionType: "both",
		}

		// Optional: balance
		if idx, ok := colMap["余额"]; ok && idx < len(record) {
			if bal, err := strconv.ParseFloat(strings.TrimSpace(record[idx]), 64); err == nil {
				user.Balance = bal
			}
		}
		// Optional: is_active
		if idx, ok := colMap["是否激活"]; ok && idx < len(record) {
			val := strings.TrimSpace(record[idx])
			if val == "否" || val == "false" || val == "0" {
				user.IsActive = false
			}
		}

		if err := db.Create(&user).Error; err != nil {
			errors = append(errors, fmt.Sprintf("第%d行: 创建用户失败 (%s)", rowNum, err.Error()))
			skipped++
			continue
		}

		// Create subscription
		subURL := utils.GenerateRandomString(64)

		// 默认值
		deviceLimit := 3
		expireTime := time.Now().AddDate(1, 0, 0) // 默认1年后过期

		// 可选: 设备限制
		if idx, ok := colMap["设备限制"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				if limit, err := strconv.Atoi(val); err == nil && limit > 0 {
					deviceLimit = limit
				}
			}
		}

		// 可选: 到期时间
		if idx, ok := colMap["到期时间"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				// 支持多种日期格式
				formats := []string{"2006-01-02", "2006/01/02", "2006-01-02 15:04:05", time.RFC3339}
				for _, format := range formats {
					if t, err := time.Parse(format, val); err == nil {
						expireTime = t
						break
					}
				}
			}
		}

		subscription := models.Subscription{
			UserID:          user.ID,
			SubscriptionURL: subURL,
			DeviceLimit:     deviceLimit,
			IsActive:        true,
			Status:          "active",
			ExpireTime:      expireTime,
		}
		if err := db.Create(&subscription).Error; err != nil {
			errors = append(errors, fmt.Sprintf("第%d行: 创建订阅失败 (%s)", rowNum, err.Error()))
			continue
		}

		imported++
	}

	utils.CreateAuditLog(c, "import_users_csv", "user", 0, fmt.Sprintf("CSV导入用户: 总计%d, 导入%d, 跳过%d", total, imported, skipped))
	utils.Success(c, gin.H{
		"total":    total,
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
	})
}

func AdminBatchNodeAction(c *gin.Context) {
	var req struct {
		IDs    []uint `json:"ids" binding:"required"`
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择节点")
		return
	}

	db := database.GetDB()
	var affected int64

	switch req.Action {
	case "enable":
		result := db.Model(&models.Node{}).Where("id IN ?", req.IDs).Update("is_active", true)
		affected = result.RowsAffected
	case "disable":
		result := db.Model(&models.Node{}).Where("id IN ?", req.IDs).Update("is_active", false)
		affected = result.RowsAffected
	case "online":
		result := db.Model(&models.Node{}).Where("id IN ?", req.IDs).Update("status", "online")
		affected = result.RowsAffected
	case "offline":
		result := db.Model(&models.Node{}).Where("id IN ?", req.IDs).Update("status", "offline")
		affected = result.RowsAffected
	case "delete":
		result := db.Where("id IN ?", req.IDs).Delete(&models.Node{})
		affected = result.RowsAffected
	default:
		utils.BadRequest(c, "不支持的操作: "+req.Action)
		return
	}

	utils.CreateAuditLog(c, "batch_node_action", "node", 0, fmt.Sprintf("批量操作节点: %s, 影响 %d 个节点", req.Action, affected))
	cache.ClearAllSubscriptionCache()
	utils.Success(c, gin.H{"affected": affected, "action": req.Action})
}

// ==================== Check-In Stats ====================

func AdminGetCheckInStats(c *gin.Context) {
	db := database.GetDB()
	today := time.Now().Format("2006-01-02")

	var todayCount, totalCount int64
	db.Model(&models.CheckIn{}).Where("DATE(created_at) = ?", today).Count(&todayCount)
	db.Model(&models.CheckIn{}).Count(&totalCount)

	var todayTotalReward float64
	db.Model(&models.CheckIn{}).Where("DATE(created_at) = ?", today).
		Select("COALESCE(SUM(amount), 0)").Scan(&todayTotalReward)

	enabled := utils.IsBoolSettingDefault("checkin_enabled", true)
	minReward := utils.GetIntSetting("checkin_min_reward", 10)
	maxReward := utils.GetIntSetting("checkin_max_reward", 50)

	utils.Success(c, gin.H{
		"today_count":        todayCount,
		"total_count":        totalCount,
		"today_total_reward": todayTotalReward,
		"settings": gin.H{
			"enabled":    enabled,
			"min_reward": minReward,
			"max_reward": maxReward,
		},
	})
}
