package handlers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ==================== Dashboard ====================

func AdminDashboard(c *gin.Context) {
	db := database.GetDB()

	var userCount, orderCount, subCount int64
	db.Model(&models.User{}).Count(&userCount)
	db.Model(&models.Order{}).Count(&orderCount)
	db.Model(&models.Subscription{}).Where("is_active = ? AND expire_time > ?", true, time.Now()).Count(&subCount)

	var revenueToday, revenueMonth float64
	today := time.Now().Format("2006-01-02")
	monthStart := time.Now().Format("2006-01") + "-01"
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) = ?", "paid", today).
		Select("COALESCE(SUM(amount), 0)").Scan(&revenueToday)
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ?", "paid", monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&revenueMonth)

	var pendingOrders, pendingTickets int64
	db.Model(&models.Order{}).Where("status = ?", "pending").Count(&pendingOrders)
	db.Model(&models.Ticket{}).Where("status IN ?", []string{"pending", "open"}).Count(&pendingTickets)

	var recentOrders []models.Order
	db.Order("created_at DESC").Limit(5).Find(&recentOrders)

	var ticketList []models.Ticket
	db.Where("status IN ?", []string{"pending", "open"}).Order("created_at DESC").Limit(5).Find(&ticketList)

	// Revenue trend (last 30 days)
	type DayStat struct {
		Date  string  `json:"date"`
		Value float64 `json:"value"`
	}
	var revenueTrend []DayStat
	thirtyDaysAgo := time.Now().AddDate(0, 0, -29).Format("2006-01-02")
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ?", "paid", thirtyDaysAgo).
		Select("DATE(payment_time) as date, COALESCE(SUM(amount), 0) as value").
		Group("DATE(payment_time)").
		Order("date ASC").
		Scan(&revenueTrend)

	// User growth (last 30 days)
	var userGrowth []DayStat
	db.Model(&models.User{}).
		Where("DATE(created_at) >= ?", thirtyDaysAgo).
		Select("DATE(created_at) as date, COUNT(*) as value").
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&userGrowth)

	utils.Success(c, gin.H{
		"total_users":          userCount,
		"active_subscriptions": subCount,
		"today_revenue":        revenueToday,
		"month_revenue":        revenueMonth,
		"pending_orders":       pendingOrders,
		"pending_tickets":      pendingTickets,
		"recent_orders":        recentOrders,
		"pending_ticket_list":  ticketList,
		"revenue_trend":        revenueTrend,
		"user_growth":          userGrowth,
	})
}

func AdminStats(c *gin.Context) {
	db := database.GetDB()

	var userCount, activeUserCount, orderCount, paidOrderCount, subCount, activeSubCount, nodeCount int64
	db.Model(&models.User{}).Count(&userCount)
	db.Model(&models.User{}).Where("is_active = ?", true).Count(&activeUserCount)
	db.Model(&models.Order{}).Count(&orderCount)
	db.Model(&models.Order{}).Where("status = ?", "paid").Count(&paidOrderCount)
	db.Model(&models.Subscription{}).Count(&subCount)
	db.Model(&models.Subscription{}).Where("is_active = ? AND expire_time > ?", true, time.Now()).Count(&activeSubCount)
	db.Model(&models.Node{}).Where("is_active = ?", true).Count(&nodeCount)

	var totalRevenue float64
	db.Model(&models.Order{}).Where("status = ?", "paid").Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)

	// New users today
	today := time.Now().Format("2006-01-02")
	var newUsersToday int64
	db.Model(&models.User{}).Where("DATE(created_at) = ?", today).Count(&newUsersToday)

	utils.Success(c, gin.H{
		"user_count":         userCount,
		"active_user_count":  activeUserCount,
		"new_users_today":    newUsersToday,
		"order_count":        orderCount,
		"paid_order_count":   paidOrderCount,
		"subscription_count": subCount,
		"active_sub_count":   activeSubCount,
		"node_count":         nodeCount,
		"total_revenue":      totalRevenue,
	})
}

// ==================== User Management ====================

func AdminListUsers(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	query := db.Model(&models.User{})
	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("username LIKE ? OR email LIKE ?", like, like)
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

	// Enrich with level name
	type UserItem struct {
		models.User
		LevelName string `json:"level_name"`
	}
	items := make([]UserItem, 0, len(users))
	// Pre-load all levels
	levelMap := make(map[uint]string)
	var levels []models.UserLevel
	db.Find(&levels)
	for _, l := range levels {
		levelMap[l.ID] = l.LevelName
	}
	for _, u := range users {
		item := UserItem{User: u}
		if u.UserLevelID != nil {
			item.LevelName = levelMap[*u.UserLevelID]
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
	db.Where("user_id = ?", id).Order("created_at DESC").Limit(10).Find(&orders)

	utils.Success(c, gin.H{"user": user, "subscription": subscription, "recent_orders": orders})
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
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}

	if len(updates) == 0 {
		utils.BadRequest(c, "没有可更新的字段")
		return
	}

	if err := db.Model(&user).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新用户失败")
		return
	}
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
	// Soft deactivate instead of hard delete
	db.Model(&user).Update("is_active", false)
	utils.SuccessMessage(c, "用户已停用")
}

func AdminToggleUserActive(c *gin.Context) {
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
	newStatus := !user.IsActive
	db.Model(&user).Update("is_active", newStatus)
	utils.Success(c, gin.H{"is_active": newStatus})
}

func AdminGetAbnormalUsers(c *gin.Context) {
	db := database.GetDB()

	// Filter by abnormal type if provided
	abnormalType := c.Query("type")

	type AbnormalUser struct {
		UserID       uint      `json:"user_id"`
		Username     string    `json:"username"`
		Email        string    `json:"email"`
		AbnormalType string    `json:"abnormal_type"`
		Details      string    `json:"details"`
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

		for _, sub := range subs {
			var user models.User
			if err := db.First(&user, sub.UserID).Error; err == nil {
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

		for _, ic := range ipCounts {
			var user models.User
			if err := db.First(&user, ic.UserID).Error; err == nil {
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

	if abnormalUsers == nil {
		abnormalUsers = []AbnormalUser{}
	}
	utils.Success(c, gin.H{"users": abnormalUsers})
}

// ==================== Order Management ====================

func AdminListOrders(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	query := db.Model(&models.Order{})
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if orderNo := c.Query("order_no"); orderNo != "" {
		query = query.Where("order_no LIKE ?", "%"+orderNo+"%")
	}

	var total int64
	query.Count(&total)

	var orders []models.Order
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&orders)

	utils.SuccessPage(c, orders, total, p.Page, p.PageSize)
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

	tx := db.Begin()
	// Calculate refund amount (use FinalAmount if available, otherwise Amount)
	refundAmount := order.Amount
	if order.FinalAmount != nil {
		refundAmount = *order.FinalAmount
	}
	// Restore user balance using GORM expression
	if err := tx.Model(&models.User{}).Where("id = ?", order.UserID).
		UpdateColumn("balance", gorm.Expr("balance + ?", refundAmount)).Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "退款失败")
		return
	}
	// Update order status
	if err := tx.Model(&order).Update("status", "refunded").Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "退款失败")
		return
	}
	tx.Commit()
	utils.SuccessMessage(c, "退款成功")
}

// ==================== Package Management ====================


func AdminListPackages(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	var total int64
	db.Model(&models.Package{}).Count(&total)

	var packages []models.Package
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&packages)

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
	if err := db.Model(&pkg).Updates(req).Error; err != nil {
		utils.InternalError(c, "更新套餐失败")
		return
	}
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

	var total int64
	query.Count(&total)

	var nodes []models.Node
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&nodes)

	utils.SuccessPage(c, nodes, total, p.Page, p.PageSize)
}

func AdminCreateNode(c *gin.Context) {
	var node models.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := database.GetDB().Create(&node).Error; err != nil {
		utils.InternalError(c, "创建节点失败")
		return
	}
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
	if err := db.Model(&node).Updates(req).Error; err != nil {
		utils.InternalError(c, "更新节点失败")
		return
	}
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

	nodes, err := services.ParseNodeLinks(content)
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
		if err := db.Create(&node).Error; err == nil {
			successCount++
		}
	}

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
	db.Model(&node).Updates(map[string]interface{}{
		"status": status, "latency": latency, "last_test": &now,
	})

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

	var total int64
	db.Model(&models.CustomNode{}).Count(&total)

	var nodes []models.CustomNode
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&nodes)

	utils.SuccessPage(c, nodes, total, p.Page, p.PageSize)
}

func AdminCreateCustomNode(c *gin.Context) {
	var node models.CustomNode
	if err := c.ShouldBindJSON(&node); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := database.GetDB().Create(&node).Error; err != nil {
		utils.InternalError(c, "创建专线节点失败")
		return
	}
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
	if err := db.Model(&node).Updates(req).Error; err != nil {
		utils.InternalError(c, "更新专线节点失败")
		return
	}
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
	db.Where("custom_node_id = ?", id).Delete(&models.UserCustomNode{})
	if err := db.Delete(&models.CustomNode{}, id).Error; err != nil {
		utils.InternalError(c, "删除专线节点失败")
		return
	}
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
	db := database.GetDB()
	// Verify custom node exists
	var node models.CustomNode
	if err := db.First(&node, id).Error; err != nil {
		utils.NotFound(c, "专线节点不存在")
		return
	}

	// Remove existing assignments and re-assign
	db.Where("custom_node_id = ?", id).Delete(&models.UserCustomNode{})
	for _, uid := range req.UserIDs {
		assignment := models.UserCustomNode{UserID: uid, CustomNodeID: uint(id)}
		db.Create(&assignment)
	}
	utils.SuccessMessage(c, "分配成功")
}

func AdminImportCustomNodeLinks(c *gin.Context) {
	var req struct {
		Links string `json:"links" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	nodes, err := services.ParseNodeLinks(req.Links)
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
		customNode := models.CustomNode{
			Name:        node.Name,
			DisplayName: node.Name,
			Protocol:    node.Type,
			Domain:      "",
			Port:        443,
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
	db.Where("custom_node_id IN ?", req.IDs).Delete(&models.UserCustomNode{})
	result := db.Where("id IN ?", req.IDs).Delete(&models.CustomNode{})
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
		"link":   node.Config,
		"name":   node.DisplayName,
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
		// Search by user email or username
		var userIDs []uint
		db.Model(&models.User{}).Where("email LIKE ? OR username LIKE ? OR CAST(id AS CHAR) = ?",
			"%"+search+"%", "%"+search+"%", search).Pluck("id", &userIDs)
		if len(userIDs) > 0 {
			query = query.Where("user_id IN ?", userIDs)
		} else {
			query = query.Where("1 = 0") // no match
		}
	}

	var total int64
	query.Count(&total)

	var subs []models.Subscription
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&subs)

	// Enrich with user email and package name
	type SubItem struct {
		models.Subscription
		UserEmail   string `json:"user_email"`
		Username    string `json:"username"`
		PackageName string `json:"package_name"`
	}
	items := make([]SubItem, 0, len(subs))
	for _, sub := range subs {
		item := SubItem{Subscription: sub}
		var user models.User
		if db.Select("email, username").First(&user, sub.UserID).Error == nil {
			item.UserEmail = user.Email
			item.Username = user.Username
		}
		if sub.PackageID != nil {
			var pkg models.Package
			if db.Select("name").First(&pkg, *sub.PackageID).Error == nil {
				item.PackageName = pkg.Name
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

	var devices []models.Device
	db.Where("subscription_id = ?", sub.ID).Find(&devices)

	result := gin.H{
		"id":                sub.ID,
		"user_id":           sub.UserID,
		"package_id":        sub.PackageID,
		"subscription_url":  sub.SubscriptionURL,
		"device_limit":      sub.DeviceLimit,
		"current_devices":   sub.CurrentDevices,
		"universal_count":   sub.UniversalCount,
		"clash_count":       sub.ClashCount,
		"is_active":         sub.IsActive,
		"status":            sub.Status,
		"expire_time":       sub.ExpireTime,
		"created_at":        sub.CreatedAt,
		"updated_at":        sub.UpdatedAt,
		"devices":           devices,
	}

	// Build full subscription URLs
	baseURL := getSubscriptionBaseURL()
	if baseURL != "" && sub.SubscriptionURL != "" {
		result["universal_url"] = fmt.Sprintf("%s/api/v1/subscribe/universal/%s", baseURL, sub.SubscriptionURL)
		result["clash_url"] = fmt.Sprintf("%s/api/v1/subscribe/%s", baseURL, sub.SubscriptionURL)
	}

	var user models.User
	if db.Select("email, username").First(&user, sub.UserID).Error == nil {
		result["user_email"] = user.Email
		result["username"] = user.Username
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
	newURL := utils.GenerateRandomString(32)

	tx := db.Begin()
	// Clear devices
	tx.Where("subscription_id = ?", sub.ID).Delete(&models.Device{})
	// Update subscription URL and reset device count
	tx.Model(&sub).Updates(map[string]interface{}{
		"subscription_url": newURL,
		"current_devices":  0,
	})
	// Log the reset
	tx.Create(&models.SubscriptionReset{
		UserID:             sub.UserID,
		SubscriptionID:     sub.ID,
		ResetType:          "admin_reset",
		Reason:             "管理员重置",
		OldSubscriptionURL: &oldURL,
		NewSubscriptionURL: &newURL,
		DeviceCountBefore:  sub.CurrentDevices,
		DeviceCountAfter:   0,
	})
	tx.Commit()

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

	newExpire := sub.ExpireTime.AddDate(0, 0, req.Days)
	db.Model(&sub).Update("expire_time", newExpire)

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
	var coupon models.Coupon
	if err := c.ShouldBindJSON(&coupon); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	adminID := c.GetUint("user_id")
	adminIDInt64 := int64(adminID)
	coupon.CreatedBy = &adminIDInt64

	if err := database.GetDB().Create(&coupon).Error; err != nil {
		utils.InternalError(c, "创建优惠券失败")
		return
	}
	utils.Success(c, coupon)
}

func AdminUpdateCoupon(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的优惠券ID")
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
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := db.Model(&coupon).Updates(req).Error; err != nil {
		utils.InternalError(c, "更新优惠券失败")
		return
	}
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
	db.Model(&ticket).Updates(req)
	utils.Success(c, ticket)
}

func AdminReplyTicket(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
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
	db.Create(&reply)
	db.Model(&models.Ticket{}).Where("id = ?", id).Update("status", "processing")
	utils.Success(c, reply)
}

func AdminListUserLevels(c *gin.Context) {
	var levels []models.UserLevel
	database.GetDB().Order("level_order ASC").Find(&levels)
	utils.Success(c, levels)
}

func AdminCreateUserLevel(c *gin.Context) {
	var level models.UserLevel
	if err := c.ShouldBindJSON(&level); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	database.GetDB().Create(&level)
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
	db.Model(&level).Updates(req)
	utils.Success(c, level)
}

func AdminDeleteUserLevel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	database.GetDB().Delete(&models.UserLevel{}, id)
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
		db.Create(&rc)
		codes = append(codes, rc)
	}
	codeStrings := make([]string, len(codes))
	for i, rc := range codes {
		codeStrings[i] = rc.Code
	}
	utils.Success(c, gin.H{"codes": codeStrings})
}

func AdminDeleteRedeemCode(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	database.GetDB().Delete(&models.RedeemCode{}, id)
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
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db := database.GetDB()
	db.Model(&models.EmailQueue{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "pending",
	})
	utils.SuccessMessage(c, "已重新加入队列")
}

func AdminGetSettings(c *gin.Context) {
	var settings []models.SystemConfig
	database.GetDB().Where("category = ? OR category IS NULL", "").Find(&settings)
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	utils.Success(c, result)
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
		db.Where("`key` = ?", k).Assign(models.SystemConfig{Key: k, Value: strVal}).FirstOrCreate(&models.SystemConfig{})
	}
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
	database.GetDB().Create(&ann)
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
	database.GetDB().Model(&ann).Updates(req)
	utils.Success(c, ann)
}

func AdminDeleteAnnouncement(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	database.GetDB().Delete(&models.Announcement{}, id)
	utils.SuccessMessage(c, "公告已删除")
}

func AdminRevenueStats(c *gin.Context) {
	db := database.GetDB()
	var totalRevenue float64
	db.Model(&models.Order{}).Where("status = ?", "paid").Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)
	var todayRevenue float64
	today := time.Now().Format("2006-01-02")
	db.Model(&models.Order{}).Where("status = ? AND DATE(payment_time) = ?", "paid", today).
		Select("COALESCE(SUM(amount), 0)").Scan(&todayRevenue)
	var monthRevenue float64
	monthStart := time.Now().Format("2006-01") + "-01"
	db.Model(&models.Order{}).Where("status = ? AND payment_time >= ?", "paid", monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&monthRevenue)
	var orderCount int64
	db.Model(&models.Order{}).Where("status = ?", "paid").Count(&orderCount)
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
		"user_count":          userCount,
		"node_count":          nodeCount,
		"active_subscriptions": activeSubCount,
		"pending_tickets":     pendingTickets,
		"pending_orders":      pendingOrders,
	})
}

func AdminCreateBackup(c *gin.Context) {
	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
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
	dstPath := filepath.Join(backupDir, fmt.Sprintf("cboard_backup_%s.db", timestamp))

	src, err := os.Open(srcPath)
	if err != nil {
		utils.InternalError(c, "打开数据库失败: "+err.Error())
		return
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		utils.InternalError(c, "创建备份文件失败: "+err.Error())
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		utils.InternalError(c, "备份失败: "+err.Error())
		return
	}

	info, _ := os.Stat(dstPath)
	utils.Success(c, gin.H{
		"filename":   filepath.Base(dstPath),
		"size":       info.Size(),
		"created_at": time.Now(),
	})
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

// ==================== Create User ====================

func AdminCreateUser(c *gin.Context) {
	var req struct {
		Username string  `json:"username" binding:"required,min=3,max=50"`
		Email    string  `json:"email" binding:"required,email"`
		Password string  `json:"password" binding:"required,min=6"`
		Balance  float64 `json:"balance"`
		IsAdmin  bool    `json:"is_admin"`
		IsActive bool    `json:"is_active"`
		Notes    string  `json:"notes"`
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
		Username:   req.Username,
		Email:      req.Email,
		Password:   hashed,
		Balance:    req.Balance,
		IsAdmin:    req.IsAdmin,
		IsActive:   req.IsActive,
		IsVerified: true,
		Theme:      "light",
		Language:   "zh-CN",
		Timezone:   "Asia/Shanghai",
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
	subURL := utils.GenerateRandomString(32)
	subscription := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: subURL,
		DeviceLimit:     3,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      time.Now(), // Expired by default, activated on purchase
	}
	db.Create(&subscription)

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
	db.Model(&user).Update("password", hashed)
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
	err := services.SendEmail(req.Email, "CBoard 测试邮件",
		"<h3>测试邮件</h3><p>如果您收到此邮件，说明 SMTP 配置正确。</p>")
	if err != nil {
		utils.InternalError(c, "发送失败: "+err.Error())
		return
	}
	utils.SuccessMessage(c, "测试邮件已发送至 "+req.Email)
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
		"device_limit": true, "is_active": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "没有可更新的字段")
		return
	}

	if err := db.Model(&sub).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新订阅失败")
		return
	}
	utils.Success(c, sub)
}

// ==================== Public Announcements ====================

func ListPublicAnnouncements(c *gin.Context) {
	db := database.GetDB()
	var items []models.Announcement
	db.Where("is_active = ?", true).Order("created_at DESC").Limit(10).Find(&items)
	utils.Success(c, items)
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
