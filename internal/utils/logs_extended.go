package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"

	"github.com/gin-gonic/gin"
)

// ==================== 订单日志 ====================

// CreateOrderLog 记录订单操作日志
func CreateOrderLog(orderID, userID uint, actionType, actionBy string, actionByUserID *uint, description string, beforeData, afterData map[string]interface{}) {
	db := database.GetDB()
	entry := models.OrderLog{
		OrderID:    orderID,
		UserID:     userID,
		ActionType: actionType,
	}
	if actionBy != "" {
		entry.ActionBy = &actionBy
	}
	if actionByUserID != nil {
		id := int64(*actionByUserID)
		entry.ActionByUserID = &id
	}
	if description != "" {
		entry.Description = &description
	}
	if beforeData != nil {
		if b, err := json.Marshal(beforeData); err == nil {
			s := string(b)
			entry.BeforeData = &s
		}
	}
	if afterData != nil {
		if b, err := json.Marshal(afterData); err == nil {
			s := string(b)
			entry.AfterData = &s
		}
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create order log: %v", err)
		}
	}()
}

// ==================== 支付日志 ====================

// CreatePaymentLog 记录支付操作日志
func CreatePaymentLog(transactionID uint, userID uint, paymentMethod, status, description string, amount float64, c *gin.Context) {
	db := database.GetDB()
	entry := models.PaymentLog{
		TransactionID: transactionID,
		UserID:        userID,
		PaymentMethod: paymentMethod,
		Amount:        amount,
		Status:        status,
	}
	if description != "" {
		entry.Description = &description
	}
	if c != nil {
		ip := GetRealClientIP(c)
		entry.IPAddress = &ip
		ua := c.GetHeader("User-Agent")
		entry.UserAgent = &ua
		location := GetIPLocation(ip)
		entry.Location = &location
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create payment log: %v", err)
		}
	}()
}

// ==================== 优惠券日志 ====================

// CreateCouponLog 记录优惠券操作日志
func CreateCouponLog(couponID, userID uint, actionType, description string, c *gin.Context) {
	db := database.GetDB()
	entry := models.CouponLog{
		CouponID:   couponID,
		UserID:     userID,
		ActionType: actionType,
	}
	if description != "" {
		entry.Description = &description
	}
	if c != nil {
		ip := GetRealClientIP(c)
		entry.IPAddress = &ip
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create coupon log: %v", err)
		}
	}()
}

// ==================== 节点日志 ====================

// CreateNodeLog 记录节点操作日志
func CreateNodeLog(nodeID uint, actionType, actionBy string, actionByUserID *uint, description string, beforeData, afterData map[string]interface{}) {
	db := database.GetDB()
	entry := models.NodeLog{
		NodeID:     nodeID,
		ActionType: actionType,
	}
	if actionBy != "" {
		entry.ActionBy = &actionBy
	}
	if actionByUserID != nil {
		id := int64(*actionByUserID)
		entry.ActionByUserID = &id
	}
	if description != "" {
		entry.Description = &description
	}
	if beforeData != nil {
		if b, err := json.Marshal(beforeData); err == nil {
			s := string(b)
			entry.BeforeData = &s
		}
	}
	if afterData != nil {
		if b, err := json.Marshal(afterData); err == nil {
			s := string(b)
			entry.AfterData = &s
		}
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create node log: %v", err)
		}
	}()
}

// ==================== 用户操作日志 ====================

// CreateUserActionLog 记录用户操作日志
func CreateUserActionLog(userID uint, actionType, module, description string, c *gin.Context) {
	db := database.GetDB()
	entry := models.UserActionLog{
		UserID:     userID,
		ActionType: actionType,
		Module:     module,
	}
	if description != "" {
		entry.Description = &description
	}
	if c != nil {
		ip := GetRealClientIP(c)
		entry.IPAddress = &ip
		ua := c.GetHeader("User-Agent")
		entry.UserAgent = &ua
		location := GetIPLocation(ip)
		entry.Location = &location
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create user action log: %v", err)
		}
	}()
}

// ==================== 管理员操作日志 ====================

// CreateAdminActionLog 记录管理员操作日志
func CreateAdminActionLog(adminID uint, actionType, module, targetType string, targetID *uint, description string, beforeData, afterData map[string]interface{}, c *gin.Context) {
	db := database.GetDB()
	entry := models.AdminActionLog{
		AdminID:    adminID,
		ActionType: actionType,
		Module:     module,
		TargetType: targetType,
	}
	if targetID != nil {
		id := int64(*targetID)
		entry.TargetID = &id
	}
	if description != "" {
		entry.Description = &description
	}
	if beforeData != nil {
		if b, err := json.Marshal(beforeData); err == nil {
			s := string(b)
			entry.BeforeData = &s
		}
	}
	if afterData != nil {
		if b, err := json.Marshal(afterData); err == nil {
			s := string(b)
			entry.AfterData = &s
		}
	}
	if c != nil {
		ip := GetRealClientIP(c)
		entry.IPAddress = &ip
		ua := c.GetHeader("User-Agent")
		entry.UserAgent = &ua
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create admin action log: %v", err)
		}
	}()
}

// ==================== 设备日志 ====================

// CreateDeviceLog 记录设备操作日志
func CreateDeviceLog(deviceID, userID uint, actionType, description string, c *gin.Context) {
	db := database.GetDB()
	entry := models.DeviceLog{
		DeviceID:   deviceID,
		UserID:     userID,
		ActionType: actionType,
	}
	if description != "" {
		entry.Description = &description
	}
	if c != nil {
		ip := GetRealClientIP(c)
		entry.IPAddress = &ip
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create device log: %v", err)
		}
	}()
}

// ==================== 工单日志 ====================

// CreateTicketLog 记录工单操作日志
func CreateTicketLog(ticketID, userID uint, actionType, actionBy string, description string) {
	db := database.GetDB()
	entry := models.TicketLog{
		TicketID:   ticketID,
		UserID:     userID,
		ActionType: actionType,
	}
	if actionBy != "" {
		entry.ActionBy = &actionBy
	}
	if description != "" {
		entry.Description = &description
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create ticket log: %v", err)
		}
	}()
}

// ==================== 邀请日志 ====================

// CreateInviteLog 记录邀请操作日志
func CreateInviteLog(inviterID, inviteeID uint, inviteCode, actionType, description string) {
	db := database.GetDB()
	entry := models.InviteLog{
		InviterID:  inviterID,
		InviteeID:  inviteeID,
		InviteCode: inviteCode,
		ActionType: actionType,
	}
	if description != "" {
		entry.Description = &description
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create invite log: %v", err)
		}
	}()
}

// ==================== 配置变更日志 ====================

// CreateConfigChangeLog 记录配置变更日志
func CreateConfigChangeLog(adminID uint, configKey, oldValue, newValue, description string, c *gin.Context) {
	db := database.GetDB()
	entry := models.ConfigChangeLog{
		AdminID:   adminID,
		ConfigKey: configKey,
	}
	if oldValue != "" {
		entry.OldValue = &oldValue
	}
	if newValue != "" {
		entry.NewValue = &newValue
	}
	if description != "" {
		entry.Description = &description
	}
	if c != nil {
		ip := GetRealClientIP(c)
		entry.IPAddress = &ip
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create config change log: %v", err)
		}
	}()
}

// ==================== 安全事件日志 ====================

// CreateSecurityLog 记录安全事件日志
func CreateSecurityLog(userID *uint, eventType, severity, description string, c *gin.Context) {
	db := database.GetDB()
	entry := models.SecurityLog{
		EventType: eventType,
		Severity:  severity,
	}
	if userID != nil {
		id := int64(*userID)
		entry.UserID = &id
	}
	if description != "" {
		entry.Description = &description
	}
	if c != nil {
		ip := GetRealClientIP(c)
		entry.IPAddress = &ip
		ua := c.GetHeader("User-Agent")
		entry.UserAgent = &ua
		location := GetIPLocation(ip)
		entry.Location = &location
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create security log: %v", err)
		}
	}()
}

// ==================== API 调用日志 ====================

// CreateAPILog 记录 API 调用日志
func CreateAPILog(userID *uint, method, path string, statusCode int, responseTime time.Duration, c *gin.Context) {
	db := database.GetDB()
	entry := models.APILog{
		Method:       method,
		Path:         path,
		StatusCode:   statusCode,
		ResponseTime: int(responseTime.Milliseconds()),
	}
	if userID != nil {
		id := int64(*userID)
		entry.UserID = &id
	}
	if c != nil {
		ip := GetRealClientIP(c)
		entry.IPAddress = &ip
		ua := c.GetHeader("User-Agent")
		entry.UserAgent = &ua
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create API log: %v", err)
		}
	}()
}

// ==================== 数据库操作日志 ====================

// CreateDatabaseLog 记录数据库操作日志
func CreateDatabaseLog(adminID uint, operation, tableName, description string, affectedRows int) {
	db := database.GetDB()
	entry := models.DatabaseLog{
		AdminID:      adminID,
		Operation:    operation,
		TableName:    tableName,
		AffectedRows: affectedRows,
	}
	if description != "" {
		entry.Description = &description
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create database log: %v", err)
		}
	}()
}

// ==================== 邮件发送日志 ====================

// CreateEmailLog 记录邮件发送日志
func CreateEmailLog(userID *uint, emailType, recipient, subject, status, errorMessage string) {
	db := database.GetDB()
	entry := models.EmailLog{
		EmailType: emailType,
		Recipient: recipient,
		Subject:   subject,
		Status:    status,
	}
	if userID != nil {
		id := int64(*userID)
		entry.UserID = &id
	}
	if errorMessage != "" {
		entry.ErrorMessage = &errorMessage
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create email log: %v", err)
		}
	}()
}

// ==================== 通知日志 ====================

// CreateNotificationLog 记录通知发送日志
func CreateNotificationLog(userID *uint, notificationType, channel, status, content string) {
	db := database.GetDB()
	entry := models.NotificationLog{
		NotificationType: notificationType,
		Channel:          channel,
		Status:           status,
	}
	if userID != nil {
		id := int64(*userID)
		entry.UserID = &id
	}
	if content != "" {
		entry.Content = &content
	}

	go func() {
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[logs] failed to create notification log: %v", err)
		}
	}()
}

// Ensure fmt is used
var _ = fmt.Sprintf
