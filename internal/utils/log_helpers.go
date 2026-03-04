package utils

import (
	"encoding/json"
	"log"

	"cboard/v2/internal/database"

	"github.com/gin-gonic/gin"
)

// LogContext 日志上下文信息
type LogContext struct {
	IPAddress *string
	UserAgent *string
	Location  *string
}

// ExtractLogContext 从 gin.Context 提取日志上下文信息
func ExtractLogContext(c *gin.Context) *LogContext {
	if c == nil {
		return nil
	}

	ip := GetRealClientIP(c)
	ua := c.GetHeader("User-Agent")
	location := GetIPLocation(ip)

	return &LogContext{
		IPAddress: &ip,
		UserAgent: &ua,
		Location:  &location,
	}
}

// ApplyLogContext 将日志上下文应用到日志实体
func ApplyLogContext(entry interface{}, ctx *LogContext) {
	if ctx == nil {
		return
	}

	// 使用反射或类型断言来设置字段
	// 这里简化处理，实际使用时需要根据具体类型设置
	type LogWithContext interface {
		SetContext(ip, ua, location *string)
	}

	if e, ok := entry.(LogWithContext); ok {
		e.SetContext(ctx.IPAddress, ctx.UserAgent, ctx.Location)
	}
}

// AsyncCreateLog 异步创建日志记录
func AsyncCreateLog(entry interface{}, logType string) {
	go func() {
		db := database.GetDB()
		if db == nil {
			return
		}
		if err := db.Create(entry).Error; err != nil {
			log.Printf("[logs] failed to create %s log: %v", logType, err)
		}
	}()
}

// MarshalToJSONString 将 map 序列化为 JSON 字符串指针
func MarshalToJSONString(data map[string]interface{}) *string {
	if data == nil {
		return nil
	}
	if b, err := json.Marshal(data); err == nil {
		s := string(b)
		return &s
	}
	return nil
}

// ToInt64Ptr 将 uint 指针转换为 int64 指针
func ToInt64Ptr(u *uint) *int64 {
	if u == nil {
		return nil
	}
	id := int64(*u)
	return &id
}

// ToStringPtr 将字符串转换为字符串指针（如果非空）
func ToStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
