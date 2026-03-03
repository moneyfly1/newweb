package middleware

import (
	"bytes"
	"cboard/v2/internal/utils"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger 记录所有请求的中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 记录请求信息
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		// 对于支付回调，记录详细信息
		if path == "/api/v1/payment/notify/alipay" || path == "/api/v1/payment/notify/epay" {
			utils.LogCallback("========================================")
			utils.LogCallback("收到支付回调请求")
			utils.LogCallback("  Method: %s", method)
			utils.LogCallback("  Path: %s", path)
			utils.LogCallback("  Client IP: %s", clientIP)
			utils.LogCallback("  User-Agent: %s", c.Request.Header.Get("User-Agent"))
			utils.LogCallback("  Content-Type: %s", c.Request.Header.Get("Content-Type"))

			// 读取请求体
			if c.Request.Body != nil {
				bodyBytes, _ := io.ReadAll(c.Request.Body)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				if len(bodyBytes) > 0 {
					utils.LogCallback("  Body: %s", string(bodyBytes))
				}
			}

			// 记录查询参数
			if len(c.Request.URL.RawQuery) > 0 {
				utils.LogCallback("  Query: %s", c.Request.URL.RawQuery)
			}
			utils.LogCallback("========================================")
		}

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()
		latency := endTime.Sub(startTime)

		// 记录响应信息
		statusCode := c.Writer.Status()

		// 对于支付相关的请求，记录详细日志
		if path == "/api/v1/payment/notify/alipay" || path == "/api/v1/payment/notify/epay" {
			utils.LogCallback("回调处理完成")
			utils.LogCallback("  Status: %d", statusCode)
			utils.LogCallback("  Latency: %v", latency)
			utils.LogCallback("========================================")
		} else if path == "/api/v1/payment/create" || path == "/api/v1/orders" {
			utils.LogPayment("[%s] %s - Status: %d, Latency: %v", method, path, statusCode, latency)
		}
	}
}
