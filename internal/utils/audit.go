package utils

import (
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateAuditLog 异步记录一条管理员操作审计日志。
func CreateAuditLog(c *gin.Context, actionType, resourceType string, resourceID uint, description string) {
	db := database.GetDB()
	ip := GetRealClientIP(c)
	ua := c.GetHeader("User-Agent")
	location := GetIPLocation(ip)

	entry := models.AuditLog{
		ActionType: actionType,
	}
	if userID := c.GetUint("user_id"); userID > 0 {
		uid := int64(userID)
		entry.UserID = &uid
	}
	if resourceType != "" {
		entry.ResourceType = &resourceType
	}
	if resourceID > 0 {
		rid := int64(resourceID)
		entry.ResourceID = &rid
	}
	if description != "" {
		entry.ActionDescription = &description
	}
	if c.Request != nil {
		entry.IPAddress = &ip
		entry.UserAgent = &ua
		entry.Location = &location
		entry.RequestMethod = &c.Request.Method
		entry.RequestPath = &c.Request.URL.Path
	}

	createLogAsync(db, &entry, "audit log")
}
