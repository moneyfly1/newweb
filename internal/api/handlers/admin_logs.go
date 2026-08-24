package handlers

import (
	"fmt"
	"strings"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AdminAuditLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.AuditLog
	var total int64
	db := database.GetDB().Model(&models.AuditLog{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminLoginLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.LoginHistory
	var total int64
	db := database.GetDB().Model(&models.LoginHistory{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminRegistrationLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.RegistrationLog
	var total int64
	db := database.GetDB().Model(&models.RegistrationLog{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminSubscriptionLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.SubscriptionLog
	var total int64
	db := database.GetDB().Model(&models.SubscriptionLog{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminBalanceLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.BalanceLog
	var total int64
	db := database.GetDB().Model(&models.BalanceLog{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminCommissionLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.CommissionLog
	var total int64
	db := database.GetDB().Model(&models.CommissionLog{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminSystemLogs(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.SystemLog
	var total int64
	db := database.GetDB().Model(&models.SystemLog{})
	if level := c.Query("level"); level != "" {
		db = db.Where("level = ?", level)
	}
	if module := c.Query("module"); module != "" {
		db = db.Where("module = ?", module)
	}
	db.Count(&total)
	db.Order("created_at DESC").Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}


// AdminClearLogs 清空指定类型的日志（全部删除）。
// type: audit / login / registration / subscription / balance / commission / system
func AdminClearLogs(c *gin.Context) {
	logType := c.Param("type")
	db := database.GetDB()

	clear := func(model interface{}) (int64, error) {
		r := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model)
		return r.RowsAffected, r.Error
	}

	var deleted int64
	var tables []string
	switch logType {
	case "audit":
		n, err := clear(&models.AuditLog{})
		if err != nil { utils.InternalError(c, "清空审计日志失败"); return }
		deleted, tables = n, []string{"audit_logs"}
	case "login":
		n1, err1 := clear(&models.LoginHistory{})
		n2, err2 := clear(&models.LoginAttempt{})
		if err1 != nil || err2 != nil { utils.InternalError(c, "清空登录日志失败"); return }
		deleted, tables = n1+n2, []string{"login_history", "login_attempts"}
	case "registration":
		n, err := clear(&models.RegistrationLog{})
		if err != nil { utils.InternalError(c, "清空注册日志失败"); return }
		deleted, tables = n, []string{"registration_logs"}
	case "subscription":
		n, err := clear(&models.SubscriptionLog{})
		if err != nil { utils.InternalError(c, "清空订阅日志失败"); return }
		deleted, tables = n, []string{"subscription_logs"}
	case "balance":
		n, err := clear(&models.BalanceLog{})
		if err != nil { utils.InternalError(c, "清空余额日志失败"); return }
		deleted, tables = n, []string{"balance_logs"}
	case "commission":
		n, err := clear(&models.CommissionLog{})
		if err != nil { utils.InternalError(c, "清空佣金日志失败"); return }
		deleted, tables = n, []string{"commission_logs"}
	case "system":
		n, err := clear(&models.SystemLog{})
		if err != nil { utils.InternalError(c, "清空系统日志失败"); return }
		deleted, tables = n, []string{"system_logs"}
	default:
		utils.BadRequest(c, "未知的日志类型")
		return
	}

	utils.CreateAuditLog(c, "clear_logs", "logs", 0, fmt.Sprintf("清空日志: %s，共删除 %d 条记录", strings.Join(tables, ","), deleted))
	utils.Success(c, gin.H{"type": logType, "tables": tables, "deleted": deleted})
}
