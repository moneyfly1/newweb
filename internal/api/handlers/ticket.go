package handlers

import (
	"fmt"
	"strconv"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// ListTickets returns paginated tickets for the current user.
func ListTickets(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()
	p := utils.GetPagination(c)

	var total int64
	query := db.Model(&models.Ticket{}).Where("user_id = ?", userID)

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	var tickets []models.Ticket
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&tickets)

	utils.SuccessPage(c, tickets, total, p.Page, p.PageSize)
}

// CreateTicket creates a new support ticket.
func CreateTicket(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Title    string `json:"title" binding:"required,max=200"`
		Content  string `json:"content" binding:"required"`
		Type     string `json:"type"`
		Priority string `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.Type == "" {
		req.Type = string(models.TicketTypeOther)
	}
	if req.Priority == "" {
		req.Priority = string(models.TicketPriorityNormal)
	}

	ticketNo := fmt.Sprintf("TK%s%s", time.Now().Format("20060102"), utils.GenerateRandomString(6))

	ticket := models.Ticket{
		TicketNo: ticketNo,
		UserID:   userID,
		Title:    req.Title,
		Content:  req.Content,
		Type:     req.Type,
		Priority: req.Priority,
		Status:   string(models.TicketStatusPending),
	}

	if err := database.GetDB().Create(&ticket).Error; err != nil {
		utils.InternalError(c, "创建工单失败")
		return
	}

	utils.Success(c, ticket)
}

// GetTicket returns a ticket with its replies. Verifies ownership.
func GetTicket(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的工单ID")
		return
	}

	db := database.GetDB()
	var ticket models.Ticket
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&ticket).Error; err != nil {
		utils.NotFound(c, "工单不存在")
		return
	}

	var replies []models.TicketReply
	db.Where("ticket_id = ?", ticket.ID).Order("created_at ASC").Find(&replies)

	utils.Success(c, gin.H{"ticket": ticket, "replies": replies})
}

// ReplyTicket adds a reply to an existing ticket.
func ReplyTicket(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的工单ID")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	db := database.GetDB()
	var ticket models.Ticket
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&ticket).Error; err != nil {
		utils.NotFound(c, "工单不存在")
		return
	}

	if ticket.Status == string(models.TicketStatusClosed) {
		utils.BadRequest(c, "工单已关闭，无法回复")
		return
	}

	reply := models.TicketReply{
		TicketID: ticket.ID,
		UserID:   userID,
		Content:  req.Content,
		IsAdmin:  false,
	}
	if err := db.Create(&reply).Error; err != nil {
		utils.InternalError(c, "回复失败")
		return
	}

	// Update ticket status to processing if it was pending
	if ticket.Status == string(models.TicketStatusPending) {
		db.Model(&ticket).Update("status", string(models.TicketStatusProcessing))
	}

	utils.Success(c, reply)
}

// CloseTicket sets the ticket status to closed.
func CloseTicket(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的工单ID")
		return
	}

	db := database.GetDB()
	var ticket models.Ticket
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&ticket).Error; err != nil {
		utils.NotFound(c, "工单不存在")
		return
	}

	now := time.Now()
	db.Model(&ticket).Updates(map[string]interface{}{
		"status":    string(models.TicketStatusClosed),
		"closed_at": &now,
	})

	utils.SuccessMessage(c, "工单已关闭")
}
