package handlers

import (
	"fmt"
	"strconv"
	"time"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
)

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
		Code               string   `json:"code" binding:"required"`
		Name               string   `json:"name"`
		Description        string   `json:"description"`
		Type               string   `json:"type" binding:"required"`
		DiscountValue      float64  `json:"discount_value"`
		MaxDiscount        *float64 `json:"max_discount"`
		MinAmount          float64  `json:"min_amount"`
		ValidFrom          string   `json:"valid_from" binding:"required"`
		ValidUntil         string   `json:"valid_until" binding:"required"`
		TotalQuantity      *int64   `json:"total_quantity"`
		MaxUsesPerUser     int      `json:"max_uses_per_user"`
		Status             string   `json:"status"`
		ApplicablePackages string   `json:"applicable_packages"`
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
		ApplicablePackages: req.ApplicablePackages,
	}
	if coupon.Status == "" {
		coupon.Status = models.CouponStatusActive
	}
	if coupon.MaxUsesPerUser == 0 {
		coupon.MaxUsesPerUser = 1
	}

	if err := database.GetDB().Create(&coupon).Error; err != nil {
		utils.InternalError(c, "创建优惠券失败")
		return
	}
	utils.CreateAuditLog(c, "create_coupon", "coupon", coupon.ID, fmt.Sprintf("创建优惠券: %s", coupon.Code))
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
		"applicable_packages": true,
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
	utils.CreateAuditLog(c, "update_coupon", "coupon", uint(id), fmt.Sprintf("更新优惠券: %s", coupon.Code))
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
	utils.CreateAuditLog(c, "delete_coupon", "coupon", uint(id), fmt.Sprintf("删除优惠券 ID: %d", id))
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
	utils.CreateAuditLog(c, "update_ticket", "ticket", uint(id), fmt.Sprintf("更新工单 ID: %d", id))
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

	var ticket models.Ticket
	if err := db.First(&ticket, id).Error; err != nil {
		utils.NotFound(c, "工单不存在")
		return
	}
	if ticket.Status == string(models.TicketStatusClosed) {
		utils.BadRequest(c, "工单已关闭，无法回复")
		return
	}

	reply := models.TicketReply{TicketID: ticket.ID, UserID: adminID, Content: req.Content, IsAdmin: true}
	if err := db.Create(&reply).Error; err != nil {
		utils.InternalError(c, "回复工单失败")
		return
	}
	if err := db.Model(&ticket).Update("status", string(models.TicketStatusProcessing)).Error; err != nil {
		utils.InternalError(c, "更新工单状态失败")
		return
	}

	var user models.User
	if err := db.First(&user, ticket.UserID).Error; err == nil && user.Email != "" {
		var replies []models.TicketReply
		db.Where("ticket_id = ?", ticket.ID).Order("created_at ASC").Find(&replies)
		historyHTML := services.BuildTicketConversationHistoryHTML(ticket, replies)
		subject, body := services.RenderEmail("ticket_reply", map[string]string{
			"username":     user.Username,
			"ticket_id":    strconv.FormatUint(uint64(ticket.ID), 10),
			"ticket_no":    ticket.TicketNo,
			"title":        ticket.Title,
			"reply":        req.Content,
			"history_html": historyHTML,
		})
		go services.QueueEmail(user.Email, subject, body, "ticket_reply")
	}

	utils.CreateAuditLog(c, "reply_ticket", "ticket", uint(id), fmt.Sprintf("回复工单 ID: %d", id))
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
	utils.CreateAuditLog(c, "create_user_level", "user_level", level.ID, fmt.Sprintf("创建用户等级: %s", level.LevelName))
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
	utils.CreateAuditLog(c, "update_user_level", "user_level", uint(id), fmt.Sprintf("更新用户等级: %s", level.LevelName))
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
	utils.CreateAuditLog(c, "delete_user_level", "user_level", uint(id), fmt.Sprintf("删除用户等级 ID: %d", id))
	utils.SuccessMessage(c, "等级已删除")
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
	utils.CreateAuditLog(c, "create_announcement", "announcement", ann.ID, fmt.Sprintf("创建公告: %s", ann.Title))
	utils.InvalidatePublicCache("public_announcements")
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
	utils.CreateAuditLog(c, "update_announcement", "announcement", uint(id), fmt.Sprintf("更新公告: %s", ann.Title))
	utils.InvalidatePublicCache("public_announcements")
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
	utils.CreateAuditLog(c, "delete_announcement", "announcement", uint(id), fmt.Sprintf("删除公告 ID: %d", id))
	utils.InvalidatePublicCache("public_announcements")
	utils.SuccessMessage(c, "公告已删除")
}

func ListPublicAnnouncements(c *gin.Context) {
	// 60s 内存缓存
	if cached := utils.GetPublicCache("public_announcements"); cached != nil {
		utils.Success(c, cached)
		return
	}
	db := database.GetDB()
	var items []models.Announcement
	db.Where("is_active = ?", true).Order("created_at DESC").Limit(10).Find(&items)
	utils.SetPublicCache("public_announcements", items)
	utils.Success(c, items)
}

// ==================== Financial Report ====================

