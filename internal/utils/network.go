package utils

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetRealClientIP extracts the real client IP from proxy headers.
// Priority: CF-Connecting-IP > X-Real-IP > X-Forwarded-For (first) > c.ClientIP()
func GetRealClientIP(c *gin.Context) string {
	if ip := firstValidIP(c.GetHeader("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if ip := firstValidIP(c.GetHeader("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		if ip := firstValidIP(parts[0]); ip != "" {
			return ip
		}
	}
	if ip := firstValidIP(c.ClientIP()); ip != "" {
		return ip
	}
	// Last resort: return raw ClientIP even if not parseable, to avoid empty string
	return c.ClientIP()
}

func firstValidIP(raw string) string {
	ip := strings.TrimSpace(raw)
	if ip == "" || net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}
