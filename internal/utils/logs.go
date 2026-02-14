package utils

import (
	"encoding/json"
	"fmt"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateRegistrationLog records a user registration event.
func CreateRegistrationLog(c *gin.Context, userID uint, username, email, inviteCode string, inviterID *uint) {
	db := database.GetDB()
	ip := GetRealClientIP(c)
	ua := c.GetHeader("User-Agent")
	location := GetIPLocation(ip)

	log := models.RegistrationLog{
		UserID:    userID,
		Username:  username,
		Email:     email,
		IPAddress: &ip,
		UserAgent: &ua,
		Location:  &location,
		Status:    "success",
	}
	if inviteCode != "" {
		log.InviteCode = &inviteCode
		source := "invite_code"
		log.RegisterSource = &source
	} else {
		source := "direct"
		log.RegisterSource = &source
	}
	if inviterID != nil {
		id := int64(*inviterID)
		log.InviterID = &id
	}

	go func() { db.Create(&log) }()
}

// CreateSubscriptionLog records a subscription change event.
func CreateSubscriptionLog(subID, userID uint, actionType, actionBy string, actionByUserID *uint, description string, beforeData, afterData map[string]interface{}) {
	db := database.GetDB()
	log := models.SubscriptionLog{
		SubscriptionID: subID,
		UserID:         userID,
		ActionType:     actionType,
	}
	if actionBy != "" {
		log.ActionBy = &actionBy
	}
	if actionByUserID != nil {
		id := int64(*actionByUserID)
		log.ActionByUserID = &id
	}
	if description != "" {
		log.Description = &description
	}
	if beforeData != nil {
		if b, err := json.Marshal(beforeData); err == nil {
			s := string(b)
			log.BeforeData = &s
		}
	}
	if afterData != nil {
		if b, err := json.Marshal(afterData); err == nil {
			s := string(b)
			log.AfterData = &s
		}
	}

	go func() { db.Create(&log) }()
}

// CreateBalanceLogEntry records a balance change event.
func CreateBalanceLogEntry(userID uint, changeType string, amount, balanceBefore, balanceAfter float64, relatedOrderID *uint, description string, c *gin.Context) {
	db := database.GetDB()
	log := models.BalanceLog{
		UserID:        userID,
		ChangeType:    changeType,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
	}
	if relatedOrderID != nil {
		id := int64(*relatedOrderID)
		log.RelatedOrderID = &id
	}
	if description != "" {
		desc := description
		log.Description = &desc
	}
	if c != nil {
		ip := GetRealClientIP(c)
		log.IPAddress = &ip
		location := GetIPLocation(ip)
		log.Location = &location
	}

	go func() { db.Create(&log) }()
}

// CreateBalanceLogSimple records a balance change without gin context (for background tasks).
func CreateBalanceLogSimple(userID uint, changeType string, amount, balanceBefore, balanceAfter float64, relatedOrderID *uint, description string) {
	CreateBalanceLogEntry(userID, changeType, amount, balanceBefore, balanceAfter, relatedOrderID, description, nil)
}

// Ensure fmt is used
var _ = fmt.Sprintf

// SysLog writes a system log entry to both stdout and the database.
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
	go func() { db.Create(&entry) }()
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
