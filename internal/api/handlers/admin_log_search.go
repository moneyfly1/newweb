package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
)

// AdminGlobalLogSearch 日志全局搜索（对齐 myweb 的全局搜索）。
//
// 后台各日志页是分表展示的（审计/登录/订阅/余额…），排查问题时往往只有一个关键字
// （用户名、邮箱、订单号、IP、订阅 token），需要一次性跨表命中。
// 这里按关键字并行查几类核心数据，返回分组结果，前端按组展示。
func AdminGlobalLogSearch(c *gin.Context) {
	kw := strings.TrimSpace(c.Query("q"))
	if len(kw) < 2 {
		utils.BadRequest(c, "请输入至少 2 个字符的关键字")
		return
	}
	db := database.GetDB()
	like := "%" + kw + "%"

	// 用户
	var users []models.User
	db.Select("id", "username", "email", "is_active", "created_at", "last_login").
		Where("username LIKE ? OR email LIKE ?", like, like).
		Order("id DESC").Limit(20).Find(&users)

	// 订单
	var orders []models.Order
	db.Where("order_no LIKE ?", like).Order("id DESC").Limit(20).Find(&orders)

	// 订阅（token / 地址）
	var subs []models.Subscription
	db.Where("subscription_url LIKE ?", like).Order("id DESC").Limit(20).Find(&subs)

	// 审计日志（动作描述 / IP）
	var auditLogs []models.AuditLog
	db.Where("action_description LIKE ? OR ip_address LIKE ?", like, like).
		Order("id DESC").Limit(30).Find(&auditLogs)

	// 登录历史（IP / UA）
	var loginLogs []models.LoginHistory
	db.Where("ip_address LIKE ? OR user_agent LIKE ?", like, like).
		Order("id DESC").Limit(30).Find(&loginLogs)

	// 订阅日志（描述）
	var subLogs []models.SubscriptionLog
	db.Where("description LIKE ?", like).Order("id DESC").Limit(30).Find(&subLogs)

	utils.Success(c, gin.H{
		"keyword":      kw,
		"users":        users,
		"orders":       orders,
		"subscriptions": subs,
		"audit_logs":   auditLogs,
		"login_logs":   loginLogs,
		"sub_logs":     subLogs,
		"total":        len(users) + len(orders) + len(subs) + len(auditLogs) + len(loginLogs) + len(subLogs),
	})
}
