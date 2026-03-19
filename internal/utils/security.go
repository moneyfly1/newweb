package utils

import (
	"fmt"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateAdminAuditLog 记录管理员操作审计日志
func CreateAdminAuditLog(c *gin.Context, action string, targetType string, targetID *uint, details string) {
	adminID, exists := c.Get("user_id")
	if !exists {
		return
	}

	ip := GetRealClientIP(c)
	ua := c.GetHeader("User-Agent")
	method := c.Request.Method
	path := c.Request.URL.Path

	uid := int64(adminID.(uint))
	var tid *int64
	if targetID != nil {
		t := int64(*targetID)
		tid = &t
	}

	log := models.AuditLog{
		UserID:            &uid,
		ActionType:        action,
		ResourceType:      &targetType,
		ResourceID:        tid,
		ActionDescription: &details,
		IPAddress:         &ip,
		UserAgent:         &ua,
		RequestMethod:     &method,
		RequestPath:       &path,
	}

	if err := database.GetDB().Create(&log).Error; err != nil {
		SysError("security", fmt.Sprintf("写入管理员审计日志失败: %v", err))
	}
}

// CreateSecurityEvent 记录安全事件
func CreateSecurityEvent(eventType string, severity string, description string, metadata map[string]interface{}) {
	// 记录到系统日志
	SysError("security", fmt.Sprintf("[%s] %s: %s", severity, eventType, description))

	// 记录到数据库
	db := database.GetDB()
	log := models.SecurityLog{
		EventType:   eventType,
		Severity:    severity,
		Description: &description,
	}
	if err := db.Create(&log).Error; err != nil {
		SysError("security", fmt.Sprintf("写入安全事件失败: %v", err))
	}

	// 注意：安全告警通知功能已实现在 services/security_alert.go
	// 可以在需要时调用 services.SendSecurityAlert()
}

// DetectSuspiciousActivity 检测可疑活动
func DetectSuspiciousActivity(c *gin.Context, activityType string) bool {
	ip := GetRealClientIP(c)
	db := database.GetDB()

	switch activityType {
	case "rapid_login_attempts":
		// 检测短时间内大量登录尝试
		var count int64
		db.Model(&models.LoginAttempt{}).
			Where("ip_address = ? AND created_at > ?", ip, time.Now().Add(-5*time.Minute)).
			Count(&count)
		if count > 20 {
			CreateSecurityEvent("rapid_login_attempts", "high",
				fmt.Sprintf("IP %s 在 5 分钟内尝试登录 %d 次", ip, count), nil)
			return true
		}

	case "subscription_enumeration":
		// 检测订阅地址枚举
		// 这个需要在 subscription handler 中调用
		return false

	case "payment_manipulation":
		// 检测支付金额篡改尝试
		return false
	}

	return false
}

// ValidateAdminAction 验证管理员操作权限
func ValidateAdminAction(c *gin.Context, action string, targetUserID uint) error {
	adminID := c.GetUint("user_id")

	// 防止管理员操作自己的账户（某些敏感操作）
	if action == "delete_user" || action == "disable_user" {
		if adminID == targetUserID {
			return fmt.Errorf("不能对自己执行此操作")
		}
	}

	// 记录审计日志
	CreateAdminAuditLog(c, action, "user", &targetUserID, fmt.Sprintf("管理员 %d 执行操作: %s", adminID, action))

	return nil
}
