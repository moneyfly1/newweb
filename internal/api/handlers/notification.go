package handlers

import (
	"strconv"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// ListNotifications returns paginated notifications for the current user.
func ListNotifications(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()
	p := utils.GetPagination(c)

	var total int64
	query := db.Model(&models.Notification{}).Where("user_id = ? AND is_active = ?", userID, true)

	if nType := c.Query("type"); nType != "" {
		query = query.Where("type = ?", nType)
	}
	if c.Query("is_read") == "true" {
		query = query.Where("is_read = ?", true)
	} else if c.Query("is_read") == "false" {
		query = query.Where("is_read = ?", false)
	}

	query.Count(&total)

	var notifications []models.Notification
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&notifications)

	utils.SuccessPage(c, notifications, total, p.Page, p.PageSize)
}

// GetUnreadCount returns the number of unread notifications.
func GetUnreadCount(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()

	var count int64
	db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ? AND is_active = ?", userID, false, true).
		Count(&count)

	utils.Success(c, gin.H{"unread_count": count})
}

// MarkNotificationRead marks a single notification as read.
func MarkNotificationRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的通知ID")
		return
	}

	db := database.GetDB()
	now := time.Now()
	result := db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		})

	if result.RowsAffected == 0 {
		utils.NotFound(c, "通知不存在")
		return
	}

	utils.SuccessMessage(c, "已标记为已读")
}

// MarkAllNotificationsRead marks all of the user's notifications as read.
func MarkAllNotificationsRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()

	now := time.Now()
	db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		})

	utils.SuccessMessage(c, "已全部标记为已读")
}

// DeleteNotification deletes a notification by ID.
func DeleteNotification(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的通知ID")
		return
	}

	db := database.GetDB()
	result := db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_active", false)

	if result.RowsAffected == 0 {
		utils.NotFound(c, "通知不存在")
		return
	}

	utils.SuccessMessage(c, "通知已删除")
}
