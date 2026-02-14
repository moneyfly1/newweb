package utils

import (
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateAuditLog records an admin action in the audit_logs table.
func CreateAuditLog(c *gin.Context, actionType, resourceType string, resourceID uint, description string) {
	db := database.GetDB()
	userID := c.GetUint("user_id")
	ip := GetRealClientIP(c)
	ua := c.GetHeader("User-Agent")
	location := GetIPLocation(ip)
	method := c.Request.Method
	path := c.Request.URL.Path

	log := models.AuditLog{
		ActionType: actionType,
	}
	if userID > 0 {
		uid := int64(userID)
		log.UserID = &uid
	}
	if resourceType != "" {
		log.ResourceType = &resourceType
	}
	if resourceID > 0 {
		rid := int64(resourceID)
		log.ResourceID = &rid
	}
	if description != "" {
		log.ActionDescription = &description
	}
	log.IPAddress = &ip
	log.UserAgent = &ua
	log.Location = &location
	log.RequestMethod = &method
	log.RequestPath = &path

	go func() { db.Create(&log) }()
}
