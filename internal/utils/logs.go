package utils

import (
	"encoding/json"
	"log"
	"regexp"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// sensitiveParamRe 匹配表单/query 中的 sign、sign_type 参数及其取值。
var sensitiveParamRe = regexp.MustCompile(`(?i)(^|[?&])(sign|sign_type)=[^&\s]*`)

// MaskSensitiveParams 将 sign/sign_type 参数的取值替换为 ***，
// 防止验签数据写入日志。URL query 与 form body 通用。
func MaskSensitiveParams(s string) string {
	if s == "" {
		return s
	}
	return sensitiveParamRe.ReplaceAllString(s, "${1}${2}=***")
}

// createLogAsync 异步写一条日志，失败时仅记录日志，不阻塞业务。
func createLogAsync(db *gorm.DB, entry interface{}, logName string) {
	if db == nil {
		return
	}
	go func() {
		if err := db.Create(entry).Error; err != nil {
			log.Printf("[logs] failed to create %s: %v", logName, err)
		}
	}()
}

// CreateRegistrationLog records a user registration event.
func CreateRegistrationLog(c *gin.Context, userID uint, username, email, inviteCode string, inviterID *uint) {
	db := database.GetDB()
	ip := GetRealClientIP(c)
	ua := c.GetHeader("User-Agent")
	location := GetIPLocation(ip)

	entry := models.RegistrationLog{
		UserID:    userID,
		Username:  username,
		Email:     email,
		IPAddress: &ip,
		UserAgent: &ua,
		Location:  &location,
		Status:    "success",
	}
	if inviteCode != "" {
		entry.InviteCode = &inviteCode
		source := "invite_code"
		entry.RegisterSource = &source
	} else {
		source := "direct"
		entry.RegisterSource = &source
	}
	if inviterID != nil {
		id := int64(*inviterID)
		entry.InviterID = &id
	}

	createLogAsync(db, &entry, "registration log")
}

// CreateSubscriptionLog records a subscription change event.
func CreateSubscriptionLog(subID, userID uint, actionType, actionBy string, actionByUserID *uint, description string, beforeData, afterData map[string]interface{}) {
	db := database.GetDB()
	entry := models.SubscriptionLog{
		SubscriptionID: subID,
		UserID:         userID,
		ActionType:     actionType,
	}
	if actionBy != "" {
		entry.ActionBy = &actionBy
	}
	if actionByUserID != nil {
		id := int64(*actionByUserID)
		entry.ActionByUserID = &id
	}
	if description != "" {
		entry.Description = &description
	}
	if beforeData != nil {
		if b, err := json.Marshal(beforeData); err == nil {
			s := string(b)
			entry.BeforeData = &s
		}
	}
	if afterData != nil {
		if b, err := json.Marshal(afterData); err == nil {
			s := string(b)
			entry.AfterData = &s
		}
	}

	createLogAsync(db, &entry, "subscription log")
}

// CreateBalanceLogEntry records a balance change event.
func CreateBalanceLogEntry(userID uint, changeType string, amount, balanceBefore, balanceAfter float64, relatedOrderID *uint, description string, c *gin.Context) {
	db := database.GetDB()
	entry := models.BalanceLog{
		UserID:        userID,
		ChangeType:    changeType,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
	}
	if relatedOrderID != nil {
		id := int64(*relatedOrderID)
		entry.RelatedOrderID = &id
	}
	if description != "" {
		desc := description
		entry.Description = &desc
	}
	if c != nil {
		ip := GetRealClientIP(c)
		entry.IPAddress = &ip
		location := GetIPLocation(ip)
		entry.Location = &location
	}

	createLogAsync(db, &entry, "balance log")
}

// CreateBalanceLogSimple records a balance change without gin context (for background tasks).
func CreateBalanceLogSimple(userID uint, changeType string, amount, balanceBefore, balanceAfter float64, relatedOrderID *uint, description string) {
	CreateBalanceLogEntry(userID, changeType, amount, balanceBefore, balanceAfter, relatedOrderID, description, nil)
}

// SysLog writes a system log entry to the database.
func SysLog(level, module, message string, detail ...string) {
	db := database.GetDB()
	if db == nil {
		return
	}
	entry := models.SystemLog{
		Level:   level,
		Module:  module,
		Message: message,
	}
	if len(detail) > 0 && detail[0] != "" {
		entry.Detail = &detail[0]
	}
	createLogAsync(db, &entry, "system log")
}

// SysInfo logs an info-level system event.
func SysInfo(module, message string) {
	SysLog("info", module, message)
}

// SysWarn logs a warning-level system event.
func SysWarn(module, message string) {
	SysLog("warn", module, message)
}

// SysError logs an error-level system event.
func SysError(module, message string, detail ...string) {
	d := ""
	if len(detail) > 0 {
		d = detail[0]
	}
	SysLog("error", module, message, d)
}
