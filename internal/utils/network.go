package utils

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// GetRealClientIP extracts the real client IP from proxy headers.
// Priority: CF-Connecting-IP > X-Real-IP > X-Forwarded-For (first) > c.ClientIP()
func GetRealClientIP(c *gin.Context) string {
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	return c.ClientIP()
}
