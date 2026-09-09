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

// listLogs 通用分页日志列表：按分页参数排序，查询失败时返回 500 而非静默空列表。
// scope 可选，用于附加过滤条件（如 system 日志的 level/module）。
func listLogs[T any](c *gin.Context, scope func(db *gorm.DB) *gorm.DB) {
	p := utils.GetPagination(c)
	db := database.GetDB().Model(new(T))
	if scope != nil {
		db = scope(db)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		utils.InternalError(c, "查询日志失败")
		return
	}

	var items []T
	if err := db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items).Error; err != nil {
		utils.InternalError(c, "查询日志失败")
		return
	}
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminAuditLogs(c *gin.Context) {
	listLogs[models.AuditLog](c, nil)
}

func AdminLoginLogs(c *gin.Context) {
	listLogs[models.LoginHistory](c, nil)
}

func AdminRegistrationLogs(c *gin.Context) {
	listLogs[models.RegistrationLog](c, nil)
}

func AdminSubscriptionLogs(c *gin.Context) {
	listLogs[models.SubscriptionLog](c, nil)
}

func AdminBalanceLogs(c *gin.Context) {
	listLogs[models.BalanceLog](c, nil)
}

func AdminCommissionLogs(c *gin.Context) {
	listLogs[models.CommissionLog](c, nil)
}

func AdminSystemLogs(c *gin.Context) {
	listLogs[models.SystemLog](c, func(db *gorm.DB) *gorm.DB {
		if level := c.Query("level"); level != "" {
			db = db.Where("level = ?", level)
		}
		if module := c.Query("module"); module != "" {
			db = db.Where("module = ?", module)
		}
		return db
	})
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
		if err != nil {
			utils.InternalError(c, "清空审计日志失败")
			return
		}
		deleted, tables = n, []string{"audit_logs"}
	case "login":
		n1, err1 := clear(&models.LoginHistory{})
		n2, err2 := clear(&models.LoginAttempt{})
		if err1 != nil || err2 != nil {
			utils.InternalError(c, "清空登录日志失败")
			return
		}
		deleted, tables = n1+n2, []string{"login_history", "login_attempts"}
	case "registration":
		n, err := clear(&models.RegistrationLog{})
		if err != nil {
			utils.InternalError(c, "清空注册日志失败")
			return
		}
		deleted, tables = n, []string{"registration_logs"}
	case "subscription":
		n, err := clear(&models.SubscriptionLog{})
		if err != nil {
			utils.InternalError(c, "清空订阅日志失败")
			return
		}
		deleted, tables = n, []string{"subscription_logs"}
	case "balance":
		n, err := clear(&models.BalanceLog{})
		if err != nil {
			utils.InternalError(c, "清空余额日志失败")
			return
		}
		deleted, tables = n, []string{"balance_logs"}
	case "commission":
		n, err := clear(&models.CommissionLog{})
		if err != nil {
			utils.InternalError(c, "清空佣金日志失败")
			return
		}
		deleted, tables = n, []string{"commission_logs"}
	case "system":
		n, err := clear(&models.SystemLog{})
		if err != nil {
			utils.InternalError(c, "清空系统日志失败")
			return
		}
		deleted, tables = n, []string{"system_logs"}
	default:
		utils.BadRequest(c, "未知的日志类型")
		return
	}

	utils.CreateAuditLog(c, "clear_logs", "logs", 0, fmt.Sprintf("清空日志: %s，共删除 %d 条记录", strings.Join(tables, ","), deleted))
	utils.Success(c, gin.H{"type": logType, "tables": tables, "deleted": deleted})
}
