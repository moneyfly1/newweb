package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
	if lineType := c.Query("line_type"); lineType != "" {
		allCustomUserIDs := db.Model(&models.UserCustomNode{}).Select("DISTINCT user_id")
		dedicatedOnlyUserIDs := db.Model(&models.UserCustomNode{}).Select("DISTINCT user_id").Where("dedicated_only = ?", true)
		legacyModeUserIDs := db.Model(&models.User{}).Select("id").Where("special_node_subscription_type IS NULL OR special_node_subscription_type = '' OR special_node_subscription_type = ?", lineTypeLegacyBoth)
		explicitDedicatedOnlyUserIDs := db.Model(&models.User{}).Select("id").Where("special_node_subscription_type = ?", lineTypeDedicatedOnly)
		explicitMixedUserIDs := db.Model(&models.User{}).Select("id").Where("special_node_subscription_type = ?", lineTypeMixed)
		explicitNormalUserIDs := db.Model(&models.User{}).Select("id").Where("special_node_subscription_type = ?", lineTypeNormal)
		switch lineType {
		case lineTypeDedicatedOnly:
			query = query.Where("user_id IN (?) OR (user_id IN (?) AND user_id IN (?))", explicitDedicatedOnlyUserIDs, legacyModeUserIDs, dedicatedOnlyUserIDs)
		case lineTypeMixed:
			query = query.Where("user_id IN (?) OR (user_id IN (?) AND user_id IN (?) AND user_id NOT IN (?))", explicitMixedUserIDs, legacyModeUserIDs, allCustomUserIDs, dedicatedOnlyUserIDs)
		case lineTypeNormal:
			query = query.Where("user_id IN (?) OR (user_id IN (?) AND user_id NOT IN (?))", explicitNormalUserIDs, legacyModeUserIDs, allCustomUserIDs)
		}
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
	type SubItem struct {
		models.Subscription
		UserEmail       string  `json:"user_email"`
		Username        string  `json:"username"`
		PackageName     string  `json:"package_name"`
		UserNotes       *string `json:"user_notes"`
		UniversalURL    string  `json:"universal_url"`
		ClashURL        string  `json:"clash_url"`
		HasCustomNodes  bool    `json:"has_custom_nodes"`
		CustomNodeCount int     `json:"custom_node_count"`
		DedicatedOnly   bool    `json:"dedicated_only"`
		LineType        string  `json:"line_type"`
	}

	// 批量查询 user 和 package，避免 N+1
	userIDs := make([]uint, 0, len(subs))
	pkgIDs := make([]int64, 0, len(subs))
	for _, sub := range subs {
		userIDs = append(userIDs, sub.UserID)
		if sub.PackageID != nil {
			pkgIDs = append(pkgIDs, *sub.PackageID)
		}
	}
	userMap := make(map[uint]models.User)
	if len(userIDs) > 0 {
		var users []models.User
		db.Select("id, email, username, notes, special_node_subscription_type").Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u
		}
	}
	customNodeSummaries := loadUserCustomNodeSummaries(db, userIDs)
	pkgMap := make(map[int64]string)
	if len(pkgIDs) > 0 {
		var pkgs []models.Package
		db.Select("id, name").Where("id IN ?", pkgIDs).Find(&pkgs)
		for _, p := range pkgs {
			pkgMap[int64(p.ID)] = p.Name
		}
	}

	items := make([]SubItem, 0, len(subs))
	for _, sub := range subs {
		item := SubItem{Subscription: sub}
		subURLs := buildSubscriptionURLs(sub.SubscriptionURL)
		item.UniversalURL, _ = subURLs["universal_url"].(string)
		item.ClashURL, _ = subURLs["clash_url"].(string)
		if u, ok := userMap[sub.UserID]; ok {
			item.UserEmail = u.Email
			item.Username = u.Username
			item.UserNotes = u.Notes
		}
		if summary, ok := customNodeSummaries[sub.UserID]; ok {
			item.HasCustomNodes = summary.Count > 0
			item.CustomNodeCount = summary.Count
			item.DedicatedOnly = summary.DedicatedOnly
		}
		if u, ok := userMap[sub.UserID]; ok {
			item.LineType = effectiveUserLineType(u.SpecialNodeSubscriptionType, item.CustomNodeCount, item.DedicatedOnly)
		} else {
			item.LineType = effectiveUserLineType("", item.CustomNodeCount, item.DedicatedOnly)
		}
		if sub.PackageID != nil {
			if name, ok := pkgMap[*sub.PackageID]; ok {
				item.PackageName = name
			}
		}
		// 仍在有效期内时，以到期时间为准纠正 status，避免显示"已过期"
		if sub.IsActive && sub.ExpireTime.After(time.Now()) {
			if time.Until(sub.ExpireTime) <= 7*24*time.Hour {
				item.Status = models.SubStatusExpiring
			} else {
				item.Status = models.SubStatusActive
			}
		}
		items = append(items, item)
	}

	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
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

	var (
		devices         []models.Device
		user            models.User
		orders          []models.Order
		balanceLogs     []models.BalanceLog
		loginHistory    []models.LoginHistory
		resets          []models.SubscriptionReset
		rechargeRecords []models.RechargeRecord
	)

	var wg sync.WaitGroup
	wg.Add(7)
	go func() { defer wg.Done(); db.Where("subscription_id = ?", sub.ID).Find(&devices) }()
	go func() { defer wg.Done(); db.First(&user, sub.UserID) }()
	go func() {
		defer wg.Done()
		db.Where("user_id = ?", sub.UserID).Order("created_at DESC").Limit(20).Find(&orders)
	}()
	go func() {
		defer wg.Done()
		db.Where("user_id = ?", sub.UserID).Order("created_at DESC").Limit(20).Find(&balanceLogs)
	}()
	go func() {
		defer wg.Done()
		db.Where("user_id = ?", sub.UserID).Order("login_time DESC").Limit(20).Find(&loginHistory)
	}()
	go func() {
		defer wg.Done()
		db.Where("user_id = ?", sub.UserID).Order("created_at DESC").Limit(20).Find(&resets)
	}()
	go func() {
		defer wg.Done()
		db.Where("user_id = ?", sub.UserID).Order("created_at DESC").Limit(20).Find(&rechargeRecords)
	}()
	wg.Wait()

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
		"recent_orders":    orders,
		"balance_logs":     balanceLogs,
		"login_history":    loginHistory,
		"resets":           resets,
		"recharge_records": rechargeRecords,
	}

	// Build full subscription URLs
	for key, value := range buildSubscriptionURLs(sub.SubscriptionURL) {
		result[key] = value
	}

	if user.ID != 0 {
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
	newURL := utils.GenerateHexToken()
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

	// 通知管理员
	var user models.User
	db.First(&user, sub.UserID)
	adminID := c.GetUint("user_id")
	var admin models.User
	db.First(&admin, adminID)
	go services.NotifyAdmin("subscription_reset", map[string]string{
		"username": user.Username,
		"reset_by": admin.Username,
	})
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

	// 原子延长（乐观锁防并发丢更新）；已过期则从现在起算
	newExpire, err := services.ExtendSubscriptionExpiry(db, sub.ID, sub.ExpireTime, req.Days)
	if err != nil {
		if errors.Is(err, services.ErrSubscriptionConflict) {
			utils.BadRequest(c, "订阅状态已变化，请刷新后重试")
			return
		}
		utils.InternalError(c, "延长订阅失败")
		return
	}
	if err := db.Model(&sub).Updates(map[string]interface{}{
		"is_active": true,
		"status":    models.SubStatusActive,
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
		"device_limit": true, "is_active": true, "expire_time": true, "protocol_filter": true,
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
				updates["status"] = models.SubStatusExpiring
			} else {
				updates["status"] = models.SubStatusActive
			}
			// 确保 is_active 为 true
			if _, hasIsActive := updates["is_active"]; !hasIsActive {
				updates["is_active"] = true
			}
		} else {
			// 已过期
			updates["status"] = models.SubStatusExpired
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
	if services.GetSiteURL() == "" {
		utils.BadRequest(c, "系统未配置域名")
		return
	}
	universalURL := services.BuildSubscriptionURL(sub.SubscriptionURL, "")
	clashURL := services.BuildSubscriptionURL(sub.SubscriptionURL, "clash")
	subject, body := services.RenderEmail("subscription", map[string]string{
		"clash_url": clashURL, "universal_url": universalURL,
		"expire_time": sub.ExpireTime.Format("2006-01-02 15:04"),
	})
	go services.QueueEmail(user.Email, subject, body, "subscription")
	utils.CreateAuditLog(c, "send_subscription_email", "subscription", sub.ID, fmt.Sprintf("向用户发送订阅邮件: %s", user.Email))
	utils.SuccessMessage(c, "订阅信息已发送至 "+user.Email)
}

// ==================== Admin Delete User (Full) ====================

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
		updates["status"] = models.SubStatusActive
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

// AdminClearSubscriptionDevices 清除订阅下全部设备记录（不触碰订阅本身与订阅链接）
func AdminClearSubscriptionDevices(c *gin.Context) {
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
	count := int64(0)
	if err := db.Model(&models.Device{}).Where("subscription_id = ?", sub.ID).Count(&count).Error; err != nil {
		utils.InternalError(c, "查询设备失败")
		return
	}
	if count > 0 {
		if err := db.Where("subscription_id = ?", sub.ID).Delete(&models.Device{}).Error; err != nil {
			utils.InternalError(c, "清除设备失败")
			return
		}
	}
	// 设备数归零
	if sub.CurrentDevices > 0 {
		if err := db.Model(&sub).UpdateColumn("current_devices", 0).Error; err != nil {
			utils.InternalError(c, "更新设备数失败")
			return
		}
	}
	adminID := c.GetUint("user_id")
	utils.CreateSubscriptionLog(sub.ID, sub.UserID, "update", "admin", &adminID, fmt.Sprintf("管理员清除设备记录 %d 台", count), nil, nil)
	utils.CreateAuditLog(c, "clear_devices", "subscription", uint(id), fmt.Sprintf("清除订阅设备记录: %d 台 (用户ID: %d)", count, sub.UserID))
	utils.Success(c, gin.H{"deleted": count})
}

// ==================== Public Announcements ====================

