package middleware

import (
	"net"
	"strings"

	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders 安全响应头
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://telegram.org; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self' https:; frame-src https://telegram.org")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Next()
	}
}

// IPWhitelist checks the client IP against the admin-configured whitelist.
// If the whitelist is empty, all IPs are allowed.
func IPWhitelist() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := utils.GetSetting("ip_whitelist")
		if raw == "" {
			c.Next()
			return
		}
		clientIP := utils.GetRealClientIP(c)
		lines := strings.Split(raw, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.Contains(line, "/") {
				// CIDR
				_, cidr, err := net.ParseCIDR(line)
				if err == nil && cidr.Contains(net.ParseIP(clientIP)) {
					c.Next()
					return
				}
			} else if line == clientIP {
				c.Next()
				return
			}
		}
		utils.Forbidden(c, "IP 不在白名单中")
		c.Abort()
	}
}
