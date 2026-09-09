package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// isPaymentNotifyPath 判断是否支付回调路径（含 epay/alipay/stripe/codepay 等全部类型）。
func isPaymentNotifyPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/payment/notify/")
}

// RequestLogger 记录支付回调与订单请求的日志。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		path := c.Request.URL.Path

		if isPaymentNotifyPath(path) {
			logCallbackRequest(c)
		}

		c.Next()

		statusCode := c.Writer.Status()
		latency := time.Since(startTime)

		if isPaymentNotifyPath(path) {
			utils.LogCallback("回调处理完成")
			utils.LogCallback("  Status: %d", statusCode)
			utils.LogCallback("  Latency: %v", latency)
			utils.LogCallback("========================================")
		} else if path == "/api/v1/payment/create" || path == "/api/v1/orders" {
			utils.LogPayment("[%s] %s - Status: %d, Latency: %v", c.Request.Method, path, statusCode, latency)
		}
	}
}

// logCallbackRequest 输出支付回调请求头与请求体（体积与敏感参数均已处理）。
// 注意：请求体会被完整读回并放回，不能截断——Stripe 等渠道校验签名需要原始 body。
func logCallbackRequest(c *gin.Context) {
	utils.LogCallback("========================================")
	utils.LogCallback("收到支付回调请求")
	utils.LogCallback("  Method: %s", c.Request.Method)
	utils.LogCallback("  Path: %s", c.Request.URL.Path)
	utils.LogCallback("  Client IP: %s", c.ClientIP())
	utils.LogCallback("  User-Agent: %s", c.Request.Header.Get("User-Agent"))
	utils.LogCallback("  Content-Type: %s", c.Request.Header.Get("Content-Type"))

	// 完整读取 body 并放回，供后续 handler 继续消费
	const bodyPreviewLen = 1024
	if c.Request.Body != nil {
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		if len(bodyBytes) > 0 {
			masked := utils.MaskSensitiveParams(string(bodyBytes))
			if len(masked) > bodyPreviewLen {
				utils.LogCallback("  Body(%d): %s... (truncated)", len(bodyBytes), truncate(masked, bodyPreviewLen))
			} else {
				utils.LogCallback("  Body(%d): %s", len(bodyBytes), masked)
			}
		}
	}

	// 记录查询参数（脱敏签名，防日志泄露验签数据）
	if raw := c.Request.URL.RawQuery; raw != "" {
		utils.LogCallback("  Query: %s", utils.MaskSensitiveParams(raw))
	}
	utils.LogCallback("========================================")
}

// truncate 安全截断字符串，避免越界。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
