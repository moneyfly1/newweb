package handlers

import (
	"cboard/v2/internal/api/middleware"
	"cboard/v2/internal/cache"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func AdminListUsers(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	query := db.Model(&models.User{})
	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		// 只在搜索词较长时才查 subscription 相关表（短词不可能是 subscription URL）
		if len(search) >= 8 {
			var resetUserIDs, subUserIDs []uint
			var swg sync.WaitGroup
			swg.Add(2)
			go func() {
				defer swg.Done()
				db.Model(&models.SubscriptionReset{}).Where("old_subscription_url LIKE ? OR new_subscription_url LIKE ?", like, like).Distinct().Pluck("user_id", &resetUserIDs)
			}()
			go func() {
				defer swg.Done()
				db.Model(&models.Subscription{}).Where("subscription_url LIKE ?", like).Pluck("user_id", &subUserIDs)
			}()
			swg.Wait()
			allIDs := append(resetUserIDs, subUserIDs...)
			if len(allIDs) > 0 {
				query = query.Where("username LIKE ? OR email LIKE ? OR notes LIKE ? OR id IN ?", like, like, like, allIDs)
			} else {
				query = query.Where("username LIKE ? OR email LIKE ? OR notes LIKE ?", like, like, like)
			}
		} else {
			query = query.Where("username LIKE ? OR email LIKE ? OR notes LIKE ?", like, like, like)
		}
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
		LevelName       string     `json:"level_name"`
		ExpireTime      *time.Time `json:"expire_time"`
		DeviceLimit     int        `json:"device_limit"`
		HasCustomNodes  bool       `json:"has_custom_nodes"`
		CustomNodeCount int        `json:"custom_node_count"`
		DedicatedOnly   bool       `json:"dedicated_only"`
		LineType        string     `json:"line_type"`
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
	userIDs := make([]uint, 0, len(users))
	if len(users) > 0 {
		for _, u := range users {
			userIDs = append(userIDs, u.ID)
		}
		var subscriptions []models.Subscription
		db.Where("user_id IN ?", userIDs).Find(&subscriptions)
		for _, sub := range subscriptions {
			subscriptionMap[sub.UserID] = sub
		}
	}
	customNodeSummaries := loadUserCustomNodeSummaries(db, userIDs)

	for _, u := range users {
		item := UserItem{User: u}
		if u.UserLevelID != nil {
			item.LevelName = levelMap[*u.UserLevelID]
		}
		if sub, ok := subscriptionMap[u.ID]; ok {
			item.ExpireTime = &sub.ExpireTime
			item.DeviceLimit = sub.DeviceLimit
		}
		if summary, ok := customNodeSummaries[u.ID]; ok {
			item.HasCustomNodes = summary.Count > 0
			item.CustomNodeCount = summary.Count
			item.DedicatedOnly = summary.DedicatedOnly
		}
		item.LineType = effectiveUserLineType(u.SpecialNodeSubscriptionType, item.CustomNodeCount, item.DedicatedOnly)
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

	var (
		orders          []models.Order
		devices         []models.Device
		resets          []models.SubscriptionReset
		balanceLogs     []models.BalanceLog
		loginHistory    []models.LoginHistory
		rechargeRecords []models.RechargeRecord
	)

	var wg sync.WaitGroup
	wg.Add(6)
	go func() { defer wg.Done(); db.Where("user_id = ?", id).Order("created_at DESC").Limit(20).Find(&orders) }()
	go func() {
		defer wg.Done()
		// 只列活跃设备：设备指纹会随客户端上报字段 / Did 变化而产生历史遗留记录，
		// 若把已停用的一并列出，同一台机器会显示成多条，管理员会误以为“设备数虚高”。
		// 真正的设备占用数以 current_devices（同样只统计 is_active）为准。
		db.Where("subscription_id = ? AND is_active = ?", subscription.ID, true).
			Order("last_access DESC").Limit(50).Find(&devices)
	}()
	go func() { defer wg.Done(); db.Where("user_id = ?", id).Order("created_at DESC").Limit(20).Find(&resets) }()
	go func() {
		defer wg.Done()
		db.Where("user_id = ?", id).Order("created_at DESC").Limit(20).Find(&balanceLogs)
	}()
	go func() {
		defer wg.Done()
		db.Where("user_id = ?", id).Order("login_time DESC").Limit(20).Find(&loginHistory)
	}()
	go func() {
		defer wg.Done()
		db.Where("user_id = ?", id).Order("created_at DESC").Limit(20).Find(&rechargeRecords)
	}()
	wg.Wait()

	// Build subscription URLs
	subURLs := buildSubscriptionURLs(subscription.SubscriptionURL)

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

func AdminGetUserCustomNodes(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.Select("id").First(&user, id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	var assignments []models.UserCustomNode
	if err := db.Where("user_id = ?", id).Order("created_at DESC").Find(&assignments).Error; err != nil {
		utils.InternalError(c, "获取用户专线分配失败")
		return
	}

	nodeIDs := make([]uint, 0, len(assignments))
	for _, assignment := range assignments {
		nodeIDs = append(nodeIDs, assignment.CustomNodeID)
	}

	nodeMap := make(map[uint]models.CustomNode, len(nodeIDs))
	if len(nodeIDs) > 0 {
		var nodes []models.CustomNode
		if err := db.Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
			utils.InternalError(c, "获取专线节点失败")
			return
		}
		for _, node := range nodes {
			nodeMap[node.ID] = node
		}
	}

	items := make([]gin.H, 0, len(assignments))
	for _, assignment := range assignments {
		node, ok := nodeMap[assignment.CustomNodeID]
		if !ok {
			continue
		}
		items = append(items, gin.H{
			"id":             assignment.ID,
			"user_id":        assignment.UserID,
			"custom_node_id": assignment.CustomNodeID,
			"expires_at":     assignment.ExpiresAt,
			"dedicated_only": assignment.DedicatedOnly,
			"limit_devices":  assignment.LimitDevices,
			"created_at":     assignment.CreatedAt,
			"updated_at":     assignment.UpdatedAt,
			"node":           node,
		})
	}

	utils.Success(c, gin.H{"items": items})
}

func AdminAssignCustomNodeToUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}

	var req struct {
		CustomNodeID  uint       `json:"custom_node_id"`
		CustomNodeIDs []uint     `json:"custom_node_ids"`
		ExpiresAt     *time.Time `json:"expires_at"`
		DedicatedOnly bool       `json:"dedicated_only"`
		LimitDevices  bool       `json:"limit_devices"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	nodeIDs := append([]uint{}, req.CustomNodeIDs...)
	if req.CustomNodeID > 0 {
		nodeIDs = append(nodeIDs, req.CustomNodeID)
	}
	nodeIDs = uniqueUintSlice(nodeIDs)
	if len(nodeIDs) == 0 {
		utils.BadRequest(c, "请选择要分配的专线节点")
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.Select("id").First(&user, id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	var existingNodeCount int64
	if err := db.Model(&models.CustomNode{}).Where("id IN ?", nodeIDs).Count(&existingNodeCount).Error; err != nil {
		utils.InternalError(c, "检查专线节点失败")
		return
	}
	if existingNodeCount != int64(len(nodeIDs)) {
		utils.BadRequest(c, "包含不存在的专线节点")
		return
	}

	userID := uint(id)
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, nodeID := range nodeIDs {
			var assignment models.UserCustomNode
			err := tx.Where("user_id = ? AND custom_node_id = ?", userID, nodeID).First(&assignment).Error
			if err == nil {
				if err := tx.Model(&assignment).Updates(map[string]interface{}{
					"expires_at":     req.ExpiresAt,
					"dedicated_only": req.DedicatedOnly,
					"limit_devices":  req.LimitDevices,
				}).Error; err != nil {
					return err
				}
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}
			if err := tx.Create(&models.UserCustomNode{
				UserID:        userID,
				CustomNodeID:  nodeID,
				ExpiresAt:     req.ExpiresAt,
				DedicatedOnly: req.DedicatedOnly,
				LimitDevices:  req.LimitDevices,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		utils.InternalError(c, "分配专线节点失败")
		return
	}

	utils.CreateAuditLog(c, "assign_user_custom_node", "user", userID, fmt.Sprintf("给用户分配 %d 个专线节点", len(nodeIDs)))
	cache.ClearAllSubscriptionCache()
	utils.SuccessMessage(c, "分配成功")
}

func AdminUnassignCustomNodeFromUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	nodeID, err := strconv.ParseUint(c.Param("nodeId"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}

	db := database.GetDB()
	result := db.Where("user_id = ? AND custom_node_id = ?", id, nodeID).Delete(&models.UserCustomNode{})
	if result.Error != nil {
		utils.InternalError(c, "解除专线分配失败")
		return
	}
	if result.RowsAffected == 0 {
		utils.NotFound(c, "专线分配不存在")
		return
	}

	utils.CreateAuditLog(c, "unassign_user_custom_node", "user", uint(id), fmt.Sprintf("解除用户专线节点 ID: %d", nodeID))
	cache.ClearAllSubscriptionCache()
	utils.SuccessMessage(c, "已解除分配")
}

func AdminUpdateUserLineType(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}

	var req struct {
		LineType string `json:"line_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	lineType := strings.TrimSpace(req.LineType)
	if lineType != lineTypeNormal && lineType != lineTypeDedicatedOnly && lineType != lineTypeMixed {
		utils.BadRequest(c, "线路类型无效")
		return
	}

	db := database.GetDB()
	userID := uint(id)
	var user models.User
	if err := db.Select("id").First(&user, userID).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	var customNodeCount int64
	if err := db.Model(&models.UserCustomNode{}).Where("user_id = ?", userID).Count(&customNodeCount).Error; err != nil {
		utils.InternalError(c, "检查专线分配失败")
		return
	}
	if lineType != lineTypeNormal && customNodeCount == 0 {
		utils.BadRequest(c, "该用户还没有分配专线节点，无法切换到专线线路")
		return
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("special_node_subscription_type", lineType).Error; err != nil {
			return err
		}
		if customNodeCount > 0 {
			return tx.Model(&models.UserCustomNode{}).
				Where("user_id = ?", userID).
				Update("dedicated_only", lineType == lineTypeDedicatedOnly).Error
		}
		return nil
	}); err != nil {
		utils.InternalError(c, "更新线路类型失败")
		return
	}

	cache.ClearAllSubscriptionCache()
	utils.CreateAuditLog(c, "update_user_line_type", "user", userID, fmt.Sprintf("更新用户线路类型: %s", lineType))
	utils.Success(c, gin.H{
		"line_type":                      lineType,
		"special_node_subscription_type": lineType,
		"has_custom_nodes":               customNodeCount > 0,
		"custom_node_count":              customNodeCount,
		"dedicated_only":                 lineType == lineTypeDedicatedOnly,
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
		if !allowed[k] && k != "expire_time" && k != "device_limit" {
			continue
		}
		if v == nil {
			// null 值剔除：GORM Updates(map) 会把 JSON null 写成 SQL NULL，
			// 非空列（如 balance）被写 NULL 会造成数据损坏
			continue
		}
		if allowed[k] {
			updates[k] = v
		} else {
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
	// 注意：expire_time/device_limit 为 null 时必须从更新中剔除，
	// 否则 GORM Updates(map) 会把订阅到期时间/设备上限写成 NULL（数据损坏）。
	if len(subscriptionUpdates) > 0 {
		for k, v := range subscriptionUpdates {
			if v == nil {
				delete(subscriptionUpdates, k)
			}
		}
		if len(subscriptionUpdates) == 0 {
			utils.CreateAuditLog(c, "update_user", "user", uint(id), fmt.Sprintf("更新用户: %s", user.Username))
			utils.Success(c, user)
			return
		}
		var subscription models.Subscription
		if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err == nil {
			// 处理 expire_time：解析失败必须**报错**，不能静默跳过
			// （静默跳过会让面板显示「已改好」而数据库里还是旧值，
			//   用户看到的就是「后台改了到期时间，客户端没变」）。
			if raw, ok := subscriptionUpdates["expire_time"]; ok {
				expireTime, perr := parseExpireTimeParam(raw)
				if perr != nil {
					utils.BadRequest(c, perr.Error())
					return
				}
				subscriptionUpdates["expire_time"] = expireTime

				// 同步更新 status
				now := time.Now()
				if expireTime.After(now) {
					// 未过期
					if time.Until(expireTime) <= 7*24*time.Hour {
						subscriptionUpdates["status"] = models.SubStatusExpiring
					} else {
						subscriptionUpdates["status"] = models.SubStatusActive
					}
					// 确保 is_active 为 true
					if _, hasIsActive := subscriptionUpdates["is_active"]; !hasIsActive {
						subscriptionUpdates["is_active"] = true
					}
				} else {
					// 已过期
					subscriptionUpdates["status"] = models.SubStatusExpired
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

// deleteUserRelatedData 在事务内删除用户全部关联数据（统一删除清单，
// 供 AdminDeleteUser 与 AdminDeleteUserFull 复用，避免删除清单不一致产生孤儿数据）
func deleteUserRelatedData(tx *gorm.DB, user *models.User) error {
	uid := user.ID
	// 先收集用户工单 ID，删除其回复
	var ticketIDs []uint
	if err := tx.Model(&models.Ticket{}).Where("user_id = ?", uid).Pluck("id", &ticketIDs).Error; err != nil {
		return err
	}
	if len(ticketIDs) > 0 {
		if err := tx.Where("ticket_id IN ?", ticketIDs).Delete(&models.TicketReply{}).Error; err != nil {
			return err
		}
	}
	cleanup := []struct {
		dest interface{}
		cond string
		args []interface{}
	}{
		{&models.PaymentTransaction{}, "user_id = ?", []interface{}{uid}},
		{&models.Notification{}, "user_id = ?", []interface{}{uid}},
		{&models.UserActivity{}, "user_id = ?", []interface{}{uid}},
		{&models.InviteCode{}, "user_id = ?", []interface{}{uid}},
		{&models.InviteRelation{}, "inviter_id = ? OR invitee_id = ?", []interface{}{uid, uid}},
		{&models.CommissionLog{}, "inviter_id = ? OR invitee_id = ?", []interface{}{uid, uid}},
		{&models.RegistrationLog{}, "user_id = ?", []interface{}{uid}},
		{&models.SubscriptionLog{}, "user_id = ?", []interface{}{uid}},
		{&models.Order{}, "user_id = ?", []interface{}{uid}},
		{&models.Device{}, "user_id = ?", []interface{}{uid}},
		{&models.SubscriptionReset{}, "user_id = ?", []interface{}{uid}},
		{&models.Subscription{}, "user_id = ?", []interface{}{uid}},
		{&models.Ticket{}, "user_id = ?", []interface{}{uid}},
		{&models.BalanceLog{}, "user_id = ?", []interface{}{uid}},
		{&models.LoginHistory{}, "user_id = ?", []interface{}{uid}},
		{&models.RechargeRecord{}, "user_id = ?", []interface{}{uid}},
		{&models.UserCustomNode{}, "user_id = ?", []interface{}{uid}},
		{&models.CheckIn{}, "user_id = ?", []interface{}{uid}},
		{&models.MysteryBoxRecord{}, "user_id = ?", []interface{}{uid}},
		{&models.CouponUsage{}, "user_id = ?", []interface{}{uid}},
	}
	for _, item := range cleanup {
		if err := tx.Where(item.cond, item.args...).Delete(item.dest).Error; err != nil {
			return err
		}
	}
	// 登录尝试与验证码按邮箱/用户名清理（用户不存在时跳过，避免空条件误删）
	if user.Email != "" || user.Username != "" {
		if err := tx.Where("username = ? OR username = ?", user.Email, user.Username).Delete(&models.LoginAttempt{}).Error; err != nil {
			return err
		}
	}
	if user.Email != "" {
		if err := tx.Where("email = ?", user.Email).Delete(&models.VerificationCode{}).Error; err != nil {
			return err
		}
	}
	return nil
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
	// 自保与管理员保护：不能删除自己，也不能删除其他管理员账号
	currentID := c.GetUint("user_id")
	if id == uint64(currentID) {
		utils.BadRequest(c, "不能删除自己的账号")
		return
	}
	if user.IsAdmin {
		utils.BadRequest(c, "不能删除管理员账号")
		return
	}
	// Send notification before deleting
	go services.NotifyUserDirect(user.Email, "account_deleted", nil)
	// Delete user and all related data in a transaction（统一删除清单）
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := deleteUserRelatedData(tx, &user); err != nil {
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
				updates["status"] = models.SubStatusActive
			} else {
				updates["status"] = models.SubStatusExpired
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
			"status":    models.SubStatusDisabled,
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
	// 清除认证缓存，使禁用/启用立即生效
	middleware.InvalidateUserCache(user.ID)
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

		if len(resetCounts) > 0 {
			userIDs := make([]uint, len(resetCounts))
			for i, rc := range resetCounts {
				userIDs[i] = rc.UserID
			}
			var users []models.User
			db.Where("id IN ?", userIDs).Find(&users)
			userMap := make(map[uint]models.User)
			for _, u := range users {
				userMap[u.ID] = u
			}
			for _, rc := range resetCounts {
				if user, ok := userMap[rc.UserID]; ok {
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
	}

	// 2. Users with too many devices (current_devices > device_limit)
	if abnormalType == "" || abnormalType == "device_limit_exceeded" {
		var subs []models.Subscription
		db.Where("current_devices > device_limit").Limit(500).Find(&subs)

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

	// 分页处理
	total := int64(len(abnormalUsers))
	p := utils.GetPagination(c)
	start := p.Offset()
	if start < 0 {
		start = 0
	}
	end := start + p.PageSize
	if start > int(total) {
		start = int(total)
	}
	if end > int(total) {
		end = int(total)
	}
	utils.SuccessPage(c, abnormalUsers[start:end], total, p.Page, p.PageSize)
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
	utils.CreateAuditLog(c, "update_user_notes", "user", uint(userID), "更新用户备注")
	utils.SuccessMessage(c, "备注已更新")
}

// ==================== GeoIP 数据库更新 ====================
//
// 历史问题：原实现是**同步**下载 6 个文件（其中 GeoLite2-City.mmdb.gz 63MB、
// ip2region_v6.xdb 36MB、v4 10MB 都在 GitHub 上），后端 60 秒超时、前端 axios
// 只有 15 秒 —— 用户点“更新 GeoIP 数据库”后必然超时，表现为“点了没反应”。
//
// 现在改为**后台异步任务**：接口立即返回，前端轮询 /update-geoip/status 看进度；
// 下载走“多镜像源 + 临时文件 + 原子替换”，单个源失败不会破坏已有可用库；
// 全部结束后统一调用 utils.ReloadGeoIP()，否则 sync.Once 缓存的旧句柄会让更新不生效。

type geoIPFileResult struct {
	File   string `json:"file"`
	OK     bool   `json:"ok"`
	Size   int64  `json:"size"`
	Source string `json:"source,omitempty"`
	Error  string `json:"error,omitempty"`
}

type geoIPUpdateState struct {
	mu         sync.Mutex
	Running    bool
	StartedAt  time.Time
	FinishedAt time.Time
	Results    []geoIPFileResult
	Message    string
}

var geoIPState = &geoIPUpdateState{}

type geoIPResource struct {
	name string
	urls []string
	gzip bool
}

// geoIPResources 待更新资源。每个文件给出多个源，按顺序尝试。
//
// 源顺序按**实测速度**排（线上 VPS 在美国，实测直连 GitHub 7.58MB/s，
// jsdelivr 2.08MB/s，ghproxy 0.35MB/s —— 直连快 20 倍以上），因此直连优先，
// 镜像仅作兜底（jsdelivr 对 >20MB 单文件有大小限制，v6 库与 mmdb 需靠直连）。
func geoIPResources() []geoIPResource {
	raw := func(p string) string { return "https://github.com/" + p }
	jsd := func(p string) string { return "https://fastly.jsdelivr.net/gh/" + p }
	return []geoIPResource{
		{name: "geoip.dat", urls: []string{
			raw("MetaCubeX/meta-rules-dat/raw/release/geoip.dat"),
			jsd("MetaCubeX/meta-rules-dat@release/geoip.dat"),
		}},
		{name: "geosite.dat", urls: []string{
			raw("MetaCubeX/meta-rules-dat/raw/release/geosite.dat"),
			jsd("MetaCubeX/meta-rules-dat@release/geosite.dat"),
		}},
		{name: "geoip.metadb", urls: []string{
			raw("MetaCubeX/meta-rules-dat/raw/release/geoip.metadb"),
			jsd("MetaCubeX/meta-rules-dat@release/geoip.metadb"),
		}},
		{name: "GeoLite2-City.mmdb", gzip: true, urls: []string{
			raw("wp-statistics/GeoLite2-City/raw/master/GeoLite2-City.mmdb.gz"),
		}},
		{name: "ip2region_v4.xdb", urls: []string{
			raw("lionsoul2014/ip2region/raw/master/data/ip2region_v4.xdb"),
			jsd("lionsoul2014/ip2region@master/data/ip2region_v4.xdb"),
		}},
		{name: "ip2region_v6.xdb", urls: []string{
			raw("lionsoul2014/ip2region/raw/master/data/ip2region_v6.xdb"),
		}},
	}
}

// AdminUpdateGeoIP 启动 GeoIP 数据库更新（后台执行，立即返回）。
func AdminUpdateGeoIP(c *gin.Context) {
	geoIPState.mu.Lock()
	if geoIPState.Running {
		geoIPState.mu.Unlock()
		utils.BadRequest(c, "GeoIP 更新任务正在执行中，请稍候")
		return
	}
	geoIPState.Running = true
	geoIPState.StartedAt = time.Now()
	geoIPState.FinishedAt = time.Time{}
	geoIPState.Results = nil
	geoIPState.Message = "任务已启动，正在后台下载…"
	geoIPState.mu.Unlock()

	if err := os.MkdirAll(filepath.Join("uploads", "config"), 0750); err != nil {
		geoIPState.mu.Lock()
		geoIPState.Running = false
		geoIPState.Message = "创建目录失败: " + err.Error()
		geoIPState.mu.Unlock()
		utils.InternalError(c, "创建 GeoIP 目录失败: "+err.Error())
		return
	}

	go runGeoIPUpdateTask()

	utils.CreateAuditLog(c, "update_geoip", "settings", 0, "更新GeoIP数据库（后台任务）")
	utils.Success(c, gin.H{
		"running": true,
		"message": "GeoIP 更新任务已在后台启动，请稍候查看结果",
	})
}

// AdminGeoIPUpdateStatus 查询 GeoIP 更新进度（前端轮询）。
func AdminGeoIPUpdateStatus(c *gin.Context) {
	geoIPState.mu.Lock()
	defer geoIPState.mu.Unlock()
	utils.Success(c, gin.H{
		"running":     geoIPState.Running,
		"started_at":  geoIPState.StartedAt,
		"finished_at": geoIPState.FinishedAt,
		"results":     geoIPState.Results,
		"message":     geoIPState.Message,
	})
}

func runGeoIPUpdateTask() {
	dir := filepath.Join("uploads", "config")
	resources := geoIPResources()
	results := make([]geoIPFileResult, len(resources))
	// 预填文件名：任务进行中时前端也能看到“哪些文件待完成”，而不是空白行
	for i, res := range resources {
		results[i] = geoIPFileResult{File: res.name}
	}

	// 并行下载：串行下载 100MB+ 太慢，且单个文件失败会拖垮整体
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i, res := range resources {
		wg.Add(1)
		go func(idx int, r geoIPResource) {
			defer wg.Done()
			item := downloadGeoIPResource(dir, r)
			mu.Lock()
			results[idx] = item
			snapshot := make([]geoIPFileResult, len(results))
			copy(snapshot, results)
			mu.Unlock()

			geoIPState.mu.Lock()
			geoIPState.Results = snapshot
			geoIPState.mu.Unlock()
		}(i, res)
	}
	wg.Wait()

	// 让新库立即生效（sync.Once 缓存不重载 = 更新无效）
	utils.ReloadGeoIP()

	okCount := 0
	for _, r := range results {
		if r.OK {
			okCount++
		}
	}
	msg := fmt.Sprintf("更新完成：成功 %d/%d，地理位置库已重载", okCount, len(results))

	geoIPState.mu.Lock()
	geoIPState.Running = false
	geoIPState.FinishedAt = time.Now()
	geoIPState.Results = results
	geoIPState.Message = msg
	geoIPState.mu.Unlock()

	utils.SysInfo("settings", "GeoIP "+msg)
}

func downloadGeoIPResource(dir string, res geoIPResource) geoIPFileResult {
	item := geoIPFileResult{File: res.name}
	// 大文件（63MB 的 mmdb.gz / 36MB 的 v6 库）需要比原 60s 更宽松的超时
	client := &http.Client{Timeout: 10 * time.Minute}
	finalPath := filepath.Join(dir, res.name)
	tmpPath := finalPath + ".tmp"

	for _, u := range res.urls {
		size, err := downloadToFile(client, u, tmpPath, res.gzip)
		if err != nil {
			item.Error = err.Error()
			_ = os.Remove(tmpPath)
			continue
		}
		// 原子替换：只有完整下载成功后才覆盖正式文件，
		// 下载中断/校验失败时旧库保持可用，不会把能用的库写坏
		if err := os.Rename(tmpPath, finalPath); err != nil {
			item.Error = err.Error()
			_ = os.Remove(tmpPath)
			continue
		}
		item.OK = true
		item.Size = size
		item.Source = u
		item.Error = ""
		return item
	}
	return item
}

func downloadToFile(client *http.Client, url, path string, isGzip bool) (int64, error) {
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %s", resp.Status)
	}

	file, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var reader io.Reader = resp.Body
	if isGzip {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return 0, err
		}
		defer gzReader.Close()
		reader = gzReader
	}
	return io.Copy(file, reader)
}

func AdminCreateUser(c *gin.Context) {
	var req struct {
		Username    string     `json:"username" binding:"required,min=3,max=50"`
		Email       string     `json:"email" binding:"required,email"`
		Password    string     `json:"password" binding:"required"`
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
	subURL := utils.GenerateHexToken()

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
		Status:          models.SubStatusActive,
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
		Password string `json:"password" binding:"required"`
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
	// 自增 token 版本号，吊销该用户全部已签发 token
	if err := db.Model(&user).UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
		utils.SysError("auth", fmt.Sprintf("更新 token 版本失败: user_id=%d err=%v", user.ID, err))
	}
	// 清除认证缓存
	middleware.InvalidateUserCache(user.ID)
	utils.CreateAuditLog(c, "reset_password", "user", uint(id), fmt.Sprintf("重置用户密码: %s", user.Username))
	utils.SuccessMessage(c, "密码已重置")
}

// ==================== Test Email ====================

func AdminDeleteUserFull(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	db := database.GetDB()
	var user models.User
	userExists := db.First(&user, id).Error == nil
	if userExists {
		// 自保与管理员保护：不能删除自己，也不能删除其他管理员账号
		currentID := c.GetUint("user_id")
		if id == uint64(currentID) {
			utils.BadRequest(c, "不能删除自己的账号")
			return
		}
		if user.IsAdmin {
			utils.BadRequest(c, "不能删除管理员账号")
			return
		}
	}

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

	// 统一删除清单（复用 deleteUserRelatedData，与普通删除保持一致）
	if userExists {
		if err := deleteUserRelatedData(tx, &user); err != nil {
			rollbackWithErr(err)
			return
		}
	} else {
		// 用户已被删除但存在孤立数据：按 id 清理可定位的关联
		ghost := &models.User{ID: uint(id)}
		if err := deleteUserRelatedData(tx, ghost); err != nil {
			rollbackWithErr(err)
			return
		}
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
		// 批量同步订阅状态：一次 IN 查询 + 两条批量 UPDATE（消除逐用户 N+1）
		var subs []models.Subscription
		if err := db.Where("user_id IN ?", req.UserIDs).Find(&subs).Error; err != nil {
			utils.InternalError(c, "查询订阅失败")
			return
		}
		now := time.Now()
		var activeIDs, expiredIDs []uint
		for _, s := range subs {
			if s.ExpireTime.After(now) {
				activeIDs = append(activeIDs, s.ID)
			} else {
				expiredIDs = append(expiredIDs, s.ID)
			}
		}
		if len(activeIDs) > 0 {
			if err := db.Model(&models.Subscription{}).Where("id IN ?", activeIDs).Updates(map[string]interface{}{
				"is_active": true, "status": models.SubStatusActive,
			}).Error; err != nil {
				utils.InternalError(c, "同步订阅状态失败")
				return
			}
		}
		if len(expiredIDs) > 0 {
			if err := db.Model(&models.Subscription{}).Where("id IN ?", expiredIDs).Updates(map[string]interface{}{
				"is_active": true, "status": models.SubStatusExpired,
			}).Error; err != nil {
				utils.InternalError(c, "同步订阅状态失败")
				return
			}
		}
	case "disable":
		result := db.Model(&models.User{}).Where("id IN ? AND is_admin = ?", req.UserIDs, false).Update("is_active", false)
		affected = result.RowsAffected
		if err := db.Model(&models.Subscription{}).Where("user_id IN ?", req.UserIDs).Updates(map[string]interface{}{
			"is_active": false, "status": models.SubStatusDisabled,
		}).Error; err != nil {
			utils.InternalError(c, "同步订阅状态失败")
			return
		}
	case "delete":
		// 与单条删除语义一致：先查用户，事务内清理全部关联数据后删除（防孤儿数据）
		var delUsers []models.User
		if err := db.Where("id IN ? AND is_admin = ?", req.UserIDs, false).Find(&delUsers).Error; err != nil {
			utils.InternalError(c, "查询用户失败")
			return
		}
		if len(delUsers) == 0 {
			utils.BadRequest(c, "没有可删除的用户（管理员账号不可批量删除）")
			return
		}
		for i := range delUsers {
			go services.NotifyUserDirect(delUsers[i].Email, "account_deleted", nil)
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			for i := range delUsers {
				if err := deleteUserRelatedData(tx, &delUsers[i]); err != nil {
					return err
				}
				if err := tx.Delete(&delUsers[i]).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			utils.SysError("admin", fmt.Sprintf("批量删除用户失败: %v", err))
			utils.InternalError(c, "批量删除用户失败")
			return
		}
		affected = int64(len(delUsers))
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
		query = query.Where("username LIKE ? OR email LIKE ? OR notes LIKE ?", like, like, like)
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
	utils.CreateAuditLog(c, "export_users_csv", "user", 0, "导出用户CSV")
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

	// 预加载库内全部 username/email 到内存判重（消除每行 Count 查询）
	existingNames := make(map[string]bool)
	existingEmails := make(map[string]bool)
	var allUsers []models.User
	db.Select("username", "email").Find(&allUsers)
	for _, u := range allUsers {
		existingNames[u.Username] = true
		existingEmails[u.Email] = true
	}

	// 收集待批量创建的用户与订阅（订阅在用户创建后回填 UserID）
	type pendingSub struct {
		userIdx int
		sub     models.Subscription
	}
	var users []models.User
	var subs []pendingSub

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

		// 内存判重（含库内已有与本次导入前的行）
		if existingEmails[email] || existingNames[username] {
			errors = append(errors, fmt.Sprintf("第%d行: 用户名或邮箱已存在 (%s / %s)", rowNum, username, email))
			skipped++
			continue
		}
		existingEmails[email] = true
		existingNames[username] = true

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

		userIdx := len(users)
		users = append(users, user)

		// 收集订阅（UserID 在批量创建用户后回填）
		subURL := utils.GenerateHexToken()
		deviceLimit := 3
		expireTime := time.Now().AddDate(1, 0, 0) // 默认1年后过期

		if idx, ok := colMap["设备限制"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				if limit, err := strconv.Atoi(val); err == nil && limit > 0 {
					deviceLimit = limit
				}
			}
		}

		if idx, ok := colMap["到期时间"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				formats := []string{"2006-01-02", "2006/01/02", "2006-01-02 15:04:05", time.RFC3339}
				for _, format := range formats {
					if t, err := time.Parse(format, val); err == nil {
						expireTime = t
						break
					}
				}
			}
		}

		subs = append(subs, pendingSub{
			userIdx: userIdx,
			sub: models.Subscription{
				SubscriptionURL: subURL,
				DeviceLimit:     deviceLimit,
				IsActive:        true,
				Status:          models.SubStatusActive,
				ExpireTime:      expireTime,
			},
		})

		imported++
	}

	// 批量写入用户（单事务批次）
	if len(users) > 0 {
		if err := db.CreateInBatches(users, 200).Error; err != nil {
			utils.InternalError(c, "批量创建用户失败: "+err.Error())
			return
		}
		// 回填订阅 UserID 并批量写入
		for i := range subs {
			subs[i].sub.UserID = users[subs[i].userIdx].ID
		}
		subModels := make([]models.Subscription, 0, len(subs))
		for _, ps := range subs {
			subModels = append(subModels, ps.sub)
		}
		if len(subModels) > 0 {
			if err := db.CreateInBatches(subModels, 200).Error; err != nil {
				errors = append(errors, "部分订阅创建失败: "+err.Error())
			}
		}
	}

	utils.CreateAuditLog(c, "import_users_csv", "user", 0, fmt.Sprintf("CSV导入用户: 总计%d, 导入%d, 跳过%d", total, imported, skipped))
	utils.Success(c, gin.H{
		"total":    total,
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
	})
}
