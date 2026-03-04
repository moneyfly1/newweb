package utils

import (
	"cboard/v2/internal/models"

	"github.com/gin-gonic/gin"
)

// ==================== 订单日志 ====================

// CreateOrderLog 记录订单操作日志
func CreateOrderLog(orderID, userID uint, actionType, actionBy string, actionByUserID *uint, description string, beforeData, afterData map[string]interface{}) {
	entry := models.OrderLog{
		OrderID:        orderID,
		UserID:         userID,
		ActionType:     actionType,
		ActionBy:       ToStringPtr(actionBy),
		ActionByUserID: ToInt64Ptr(actionByUserID),
		Description:    ToStringPtr(description),
		BeforeData:     MarshalToJSONString(beforeData),
		AfterData:      MarshalToJSONString(afterData),
	}
	AsyncCreateLog(&entry, "order")
}

// ==================== 支付日志 ====================

// CreatePaymentLog 记录支付操作日志
func CreatePaymentLog(transactionID, userID uint, paymentMethod, status, description string, amount float64, c *gin.Context) {
	ctx := ExtractLogContext(c)
	entry := models.PaymentLog{
		TransactionID: transactionID,
		UserID:        userID,
		PaymentMethod: paymentMethod,
		Amount:        amount,
		Status:        status,
		Description:   ToStringPtr(description),
	}
	if ctx != nil {
		entry.IPAddress = ctx.IPAddress
		entry.UserAgent = ctx.UserAgent
		entry.Location = ctx.Location
	}
	AsyncCreateLog(&entry, "payment")
}

// ==================== 优惠券日志 ====================

// CreateCouponLog 记录优惠券操作日志
func CreateCouponLog(couponID, userID uint, actionType, description string, c *gin.Context) {
	ctx := ExtractLogContext(c)
	entry := models.CouponLog{
		CouponID:    couponID,
		UserID:      userID,
		ActionType:  actionType,
		Description: ToStringPtr(description),
	}
	if ctx != nil {
		entry.IPAddress = ctx.IPAddress
	}
	AsyncCreateLog(&entry, "coupon")
}

// ==================== 节点日志 ====================

// CreateNodeLog 记录节点操作日志
func CreateNodeLog(nodeID uint, actionType, actionBy string, actionByUserID *uint, description string, beforeData, afterData map[string]interface{}) {
	entry := models.NodeLog{
		NodeID:         nodeID,
		ActionType:     actionType,
		ActionBy:       ToStringPtr(actionBy),
		ActionByUserID: ToInt64Ptr(actionByUserID),
		Description:    ToStringPtr(description),
		BeforeData:     MarshalToJSONString(beforeData),
		AfterData:      MarshalToJSONString(afterData),
	}
	AsyncCreateLog(&entry, "node")
}

// ==================== 用户操作日志 ====================

// CreateUserActionLog 记录用户操作日志
func CreateUserActionLog(userID uint, actionType, module, description string, c *gin.Context) {
	ctx := ExtractLogContext(c)
	entry := models.UserActionLog{
		UserID:      userID,
		ActionType:  actionType,
		Module:      module,
		Description: ToStringPtr(description),
	}
	if ctx != nil {
		entry.IPAddress = ctx.IPAddress
		entry.UserAgent = ctx.UserAgent
		entry.Location = ctx.Location
	}
	AsyncCreateLog(&entry, "user_action")
}

// ==================== 管理员操作日志 ====================

// CreateAdminActionLog 记录管理员操作日志
func CreateAdminActionLog(adminID uint, actionType, module, targetType string, targetID *uint, description string, beforeData, afterData map[string]interface{}, c *gin.Context) {
	ctx := ExtractLogContext(c)
	entry := models.AdminActionLog{
		AdminID:     adminID,
		ActionType:  actionType,
		Module:      module,
		TargetType:  targetType,
		TargetID:    ToInt64Ptr(targetID),
		Description: ToStringPtr(description),
		BeforeData:  MarshalToJSONString(beforeData),
		AfterData:   MarshalToJSONString(afterData),
	}
	if ctx != nil {
		entry.IPAddress = ctx.IPAddress
		entry.UserAgent = ctx.UserAgent
	}
	AsyncCreateLog(&entry, "admin_action")
}

// ==================== 设备日志 ====================

// CreateDeviceLog 记录设备操作日志
func CreateDeviceLog(deviceID, userID uint, actionType, description string, c *gin.Context) {
	ctx := ExtractLogContext(c)
	entry := models.DeviceLog{
		DeviceID:    deviceID,
		UserID:      userID,
		ActionType:  actionType,
		Description: ToStringPtr(description),
	}
	if ctx != nil {
		entry.IPAddress = ctx.IPAddress
	}
	AsyncCreateLog(&entry, "device")
}

// ==================== 工单日志 ====================

// CreateTicketLog 记录工单操作日志
func CreateTicketLog(ticketID, userID uint, actionType, actionBy, description string) {
	entry := models.TicketLog{
		TicketID:    ticketID,
		UserID:      userID,
		ActionType:  actionType,
		ActionBy:    ToStringPtr(actionBy),
		Description: ToStringPtr(description),
	}
	AsyncCreateLog(&entry, "ticket")
}

// ==================== 邀请日志 ====================

// CreateInviteLog 记录邀请操作日志
func CreateInviteLog(inviterID, inviteeID uint, inviteCode, actionType, description string) {
	entry := models.InviteLog{
		InviterID:   inviterID,
		InviteeID:   inviteeID,
		InviteCode:  inviteCode,
		ActionType:  actionType,
		Description: ToStringPtr(description),
	}
	AsyncCreateLog(&entry, "invite")
}

// ==================== 配置变更日志 ====================

// CreateConfigChangeLog 记录配置变更日志
func CreateConfigChangeLog(adminID uint, configKey, oldValue, newValue, description string, c *gin.Context) {
	ctx := ExtractLogContext(c)
	entry := models.ConfigChangeLog{
		AdminID:     adminID,
		ConfigKey:   configKey,
		OldValue:    ToStringPtr(oldValue),
		NewValue:    ToStringPtr(newValue),
		Description: ToStringPtr(description),
	}
	if ctx != nil {
		entry.IPAddress = ctx.IPAddress
	}
	AsyncCreateLog(&entry, "config_change")
}

// ==================== 安全事件日志 ====================

// CreateSecurityLog 记录安全事件日志
func CreateSecurityLog(userID *uint, eventType, severity, description string, c *gin.Context) {
	ctx := ExtractLogContext(c)
	entry := models.SecurityLog{
		UserID:      ToInt64Ptr(userID),
		EventType:   eventType,
		Severity:    severity,
		Description: ToStringPtr(description),
	}
	if ctx != nil {
		entry.IPAddress = ctx.IPAddress
		entry.UserAgent = ctx.UserAgent
		entry.Location = ctx.Location
	}
	AsyncCreateLog(&entry, "security")
}

// ==================== API 调用日志 ====================

// CreateAPILog 记录 API 调用日志（简化版，用于关键 API）
func CreateAPILog(userID *uint, method, path string, statusCode, responseTimeMs int, c *gin.Context) {
	ctx := ExtractLogContext(c)
	entry := models.APILog{
		UserID:       ToInt64Ptr(userID),
		Method:       method,
		Path:         path,
		StatusCode:   statusCode,
		ResponseTime: responseTimeMs,
	}
	if ctx != nil {
		entry.IPAddress = ctx.IPAddress
		entry.UserAgent = ctx.UserAgent
	}
	AsyncCreateLog(&entry, "api")
}

// ==================== 数据库操作日志 ====================

// CreateDatabaseLog 记录数据库操作日志
func CreateDatabaseLog(adminID uint, operation, tableName, description string, affectedRows int) {
	entry := models.DatabaseLog{
		AdminID:      adminID,
		Operation:    operation,
		TableName:    tableName,
		AffectedRows: affectedRows,
		Description:  ToStringPtr(description),
	}
	AsyncCreateLog(&entry, "database")
}

// ==================== 邮件发送日志 ====================

// CreateEmailLog 记录邮件发送日志
func CreateEmailLog(userID *uint, emailType, recipient, subject, status, errorMessage string) {
	entry := models.EmailLog{
		UserID:       ToInt64Ptr(userID),
		EmailType:    emailType,
		Recipient:    recipient,
		Subject:      subject,
		Status:       status,
		ErrorMessage: ToStringPtr(errorMessage),
	}
	AsyncCreateLog(&entry, "email")
}

// ==================== 通知日志 ====================

// CreateNotificationLog 记录通知发送日志
func CreateNotificationLog(userID *uint, notificationType, channel, status, content string) {
	entry := models.NotificationLog{
		UserID:           ToInt64Ptr(userID),
		NotificationType: notificationType,
		Channel:          channel,
		Status:           status,
		Content:          ToStringPtr(content),
	}
	AsyncCreateLog(&entry, "notification")
}
