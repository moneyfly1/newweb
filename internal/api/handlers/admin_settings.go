package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
)

func AdminListRedeemCodes(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.RedeemCode
	var total int64
	db := database.GetDB().Model(&models.RedeemCode{})
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminCreateRedeemCodes(c *gin.Context) {
	var req struct {
		Code      string  `json:"code"`
		Name      string  `json:"name"`
		Type      string  `json:"type" binding:"required"`
		Value     float64 `json:"value" binding:"required"`
		PackageID *uint   `json:"package_id"`
		Quantity  int     `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	adminID := c.GetUint("user_id")
	db := database.GetDB()
	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}
	name := req.Name
	if name == "" {
		name = req.Type + " 卡密"
	}
	records := make([]models.RedeemCode, 0, qty)
	for i := 0; i < qty; i++ {
		code := req.Code
		if code == "" || qty > 1 {
			code = utils.GenerateRandomString(16)
		}
		records = append(records, models.RedeemCode{
			Code:      code,
			Name:      name,
			Type:      req.Type,
			Value:     req.Value,
			PackageID: req.PackageID,
			Status:    "unused",
			CreatedBy: adminID,
		})
	}
	// 批量写入（单事务），避免上千张卡密逐条提交导致慢与半批失败
	if err := db.CreateInBatches(records, 200).Error; err != nil {
		utils.InternalError(c, "创建卡密失败")
		return
	}
	codes := make([]string, len(records))
	for i, rc := range records {
		codes[i] = rc.Code
	}
	utils.CreateAuditLog(c, "create_redeem_codes", "redeem_code", 0, "批量生成兑换码")
	utils.Success(c, gin.H{"codes": codes})
}

func AdminDeleteRedeemCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	if err := database.GetDB().Delete(&models.RedeemCode{}, id).Error; err != nil {
		utils.InternalError(c, "删除卡密失败")
		return
	}
	utils.CreateAuditLog(c, "delete_redeem_code", "redeem_code", uint(id), fmt.Sprintf("删除兑换码 ID: %d", id))
	utils.SuccessMessage(c, "卡密已删除")
}

func AdminListEmailQueue(c *gin.Context) {
	p := utils.GetPagination(c)
	var items []models.EmailQueue
	var total int64
	db := database.GetDB().Model(&models.EmailQueue{})
	status := c.Query("status")
	if status != "" {
		db = db.Where("status = ?", status)
	}
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminRetryEmail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	db := database.GetDB()
	// 校验邮件记录存在，避免「重试不存在的 ID 也提示成功」
	var count int64
	if err := db.Model(&models.EmailQueue{}).Where("id = ?", id).Count(&count).Error; err != nil {
		utils.InternalError(c, "查询邮件失败")
		return
	}
	if count == 0 {
		utils.NotFound(c, "邮件记录不存在")
		return
	}
	if err := db.Model(&models.EmailQueue{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "pending",
	}).Error; err != nil {
		utils.InternalError(c, "重试邮件失败")
		return
	}
	utils.CreateAuditLog(c, "retry_email", "email_queue", uint(id), fmt.Sprintf("重试发送邮件 ID: %d", id))
	utils.SuccessMessage(c, "已重新加入队列")
}

func AdminDeleteEmail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的邮件ID")
		return
	}
	if err := database.GetDB().Delete(&models.EmailQueue{}, id).Error; err != nil {
		utils.InternalError(c, "删除失败")
		return
	}
	utils.CreateAuditLog(c, "delete_email", "email_queue", uint(id), fmt.Sprintf("删除邮件队列 ID: %d", id))
	utils.SuccessMessage(c, "邮件记录已删除")
}

// sensitiveSettingKey 判断是否为敏感配置键（回显时需掩码）
func sensitiveSettingKey(key string) bool {
	lower := strings.ToLower(key)
	for _, fragment := range []string{
		"secret", "password", "passwd", "private_key", "privatekey",
		"token", "webhook", "app_secret", "bot_key",
	} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

// maskSettingValue 掩码敏感配置值，仅保留末 4 位
func maskSettingValue(key, value string) string {
	if !sensitiveSettingKey(key) || value == "" {
		return value
	}
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}

// isMaskedSettingValue 判断提交的值是否为掩码占位（跳过更新以保留原值）
func isMaskedSettingValue(value string) bool {
	return strings.HasPrefix(value, "****")
}

func AdminGetSettings(c *gin.Context) {
	var settings []models.SystemConfig
	database.GetDB().Where("category = ? OR category IS NULL", "").Find(&settings)
	result := make(map[string]string)
	for _, s := range settings {
		// 密钥类配置掩码回显，防止支付私钥/SMTP 密码/机器人 Token 等明文常驻管理端
		result[s.Key] = maskSettingValue(s.Key, s.Value)
	}
	utils.Success(c, result)
}

func AdminUpdateSettings(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	db := database.GetDB()
	for k, v := range req {
		// 键名白名单校验：仅接受小写字母/数字/下划线组成的合法键，
		// 排除内部保留前缀（如 subscription_fetch_cache_* 缓存键），防止写入任意 SystemConfig 行
		if !validSettingKey(k) {
			utils.BadRequest(c, fmt.Sprintf("非法的设置键名: %s", k))
			return
		}
		strVal := fmt.Sprintf("%v", v)
		// 值长度限制（防止注入超长文本）
		if len(strVal) > 4096 {
			utils.BadRequest(c, fmt.Sprintf("设置项 %s 的值过长", k))
			return
		}
		// 掩码占位值（**** 前缀）跳过更新，避免用掩码覆盖真实密钥
		if isMaskedSettingValue(strVal) && sensitiveSettingKey(k) {
			continue
		}
		// 使用 Updates(map) 以避免触发全字段覆盖，仅更新 value 字段
		// 这样可以保留原有的 category, display_name 等信息
		result := db.Model(&models.SystemConfig{}).Where("`key` = ?", k).Updates(map[string]interface{}{"value": strVal})
		if result.Error == nil && result.RowsAffected == 0 {
			// 如果记录不存在，则创建新记录，默认 category 为空
			db.Create(&models.SystemConfig{Key: k, Value: strVal, Category: ""})
		}
	}
	utils.CreateAuditLog(c, "update_settings", "settings", 0, "更新系统设置")
	utils.InvalidateSettingsCache()
	utils.SuccessMessage(c, "设置已更新")
}

// validSettingKey 校验设置键名：小写字母开头，仅含小写字母/数字/下划线，
// 且不属于内部保留前缀（缓存类键不可经设置接口覆盖）
func validSettingKey(k string) bool {
	if k == "" || len(k) > 100 {
		return false
	}
	for i, ch := range k {
		if ch >= 'a' && ch <= 'z' {
			continue
		}
		if ch >= '0' && ch <= '9' && i > 0 {
			continue
		}
		if ch == '_' && i > 0 {
			continue
		}
		return false
	}
	if strings.HasPrefix(k, "subscription_fetch_cache_") {
		return false
	}
	return true
}

func AdminCleanOldLogs(c *gin.Context) {
	retentionDays := utils.GetIntSetting("log_retention_days", 90)
	if retentionDays <= 0 {
		utils.BadRequest(c, "日志保留天数未配置或无效")
		return
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	db := database.GetDB()
	type result struct {
		Table   string `json:"table"`
		Deleted int64  `json:"deleted"`
	}
	results := make([]result, 0)
	totalDeleted := int64(0)

	logTables := []struct {
		model interface{}
		name  string
	}{
		{&models.AuditLog{}, "audit_logs"},
		{&models.RegistrationLog{}, "registration_logs"},
		{&models.SubscriptionLog{}, "subscription_logs"},
		{&models.BalanceLog{}, "balance_logs"},
		{&models.CommissionLog{}, "commission_logs"},
		{&models.SystemLog{}, "system_logs"},
		{&models.OrderLog{}, "order_logs"},
		{&models.PaymentLog{}, "payment_logs"},
		{&models.CouponLog{}, "coupon_logs"},
		{&models.NodeLog{}, "node_logs"},
		{&models.UserActionLog{}, "user_action_logs"},
		{&models.AdminActionLog{}, "admin_action_logs"},
		{&models.DeviceLog{}, "device_logs"},
		{&models.TicketLog{}, "ticket_logs"},
		{&models.InviteLog{}, "invite_logs"},
		{&models.ConfigChangeLog{}, "config_change_logs"},
		{&models.SecurityLog{}, "security_logs"},
		{&models.APILog{}, "api_logs"},
		{&models.DatabaseLog{}, "database_logs"},
		{&models.EmailLog{}, "email_logs"},
		{&models.NotificationLog{}, "notification_logs"},
		{&models.LoginHistory{}, "login_history"},
		{&models.UserActivity{}, "user_activities"},
		{&models.LoginAttempt{}, "login_attempts"},
		{&models.VerificationAttempt{}, "verification_attempts"},
	}

	for _, t := range logTables {
		r := db.Where("created_at < ?", cutoff).Delete(t.model)
		results = append(results, result{Table: t.name, Deleted: r.RowsAffected})
		totalDeleted += r.RowsAffected
	}

	utils.CreateAuditLog(c, "clean_logs", "logs", 0, fmt.Sprintf("清理 %d 天前的日志，共删除 %d 条记录", retentionDays, totalDeleted))
	utils.Success(c, gin.H{
		"retention_days": retentionDays,
		"cutoff":         cutoff.Format("2006-01-02 15:04:05"),
		"total_deleted":  totalDeleted,
		"details":        results,
	})
}

func AdminBackfillLocations(c *gin.Context) {
	db := database.GetDB()
	backfilled := map[string]int64{}

	updateTable := func(tableName string) error {
		rows, err := db.Table(tableName).Select("id, ip_address").Where("(location IS NULL OR location = '') AND ip_address IS NOT NULL AND ip_address != ''").Rows()
		if err != nil {
			return err
		}
		defer rows.Close()

		var count int64
		for rows.Next() {
			var id uint
			var ip string
			if err := rows.Scan(&id, &ip); err != nil {
				continue
			}
			location := utils.GetIPLocation(ip)
			if location == "" {
				continue
			}
			if err := db.Table(tableName).Where("id = ?", id).Update("location", location).Error; err == nil {
				count++
			}
		}
		backfilled[tableName] = count
		return nil
	}

	for _, tableName := range []string{"login_history", "user_activities", "registration_logs", "subscription_logs", "balance_logs"} {
		if err := updateTable(tableName); err != nil {
			utils.InternalError(c, "回填 "+tableName+" 失败: "+err.Error())
			return
		}
	}

	utils.CreateAuditLog(c, "backfill_locations", "settings", 0, "回填IP位置信息")
	utils.Success(c, gin.H{
		"backfilled": backfilled,
		"message":    "历史地区数据回填完成",
	})
}

// ==================== Create User ====================

func AdminSendTestEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	subject, body := services.RenderEmail("test", map[string]string{})
	err := services.SendEmail(req.Email, subject, body)
	if err != nil {
		utils.InternalError(c, "发送失败: "+err.Error())
		return
	}
	utils.CreateAuditLog(c, "send_test_email", "settings", 0, "发送测试邮件")
	utils.SuccessMessage(c, "测试邮件已发送至 "+req.Email)
}

func AdminTestTelegram(c *gin.Context) {
	if err := services.SendTestTelegram(); err != nil {
		utils.InternalError(c, "发送失败: "+err.Error())
		return
	}
	utils.CreateAuditLog(c, "test_telegram", "settings", 0, "测试Telegram通知")
	utils.SuccessMessage(c, "Telegram 测试消息已发送")
}

func AdminTestBark(c *gin.Context) {
	if err := services.SendTestBark(); err != nil {
		utils.InternalError(c, "发送失败: "+err.Error())
		return
	}
	utils.CreateAuditLog(c, "test_bark", "settings", 0, "测试Bark通知")
	utils.SuccessMessage(c, "Bark 测试消息已发送")
}

// ==================== Update Subscription ====================

func AdminGetProtocolFilter(c *gin.Context) {
	db := database.GetDB()
	result := make(map[string][]string)
	for _, key := range []string{"clash_protocols", "universal_protocols"} {
		var cfg models.SystemConfig
		if err := db.Where("category = ? AND `key` = ?", "protocol_filter", key).First(&cfg).Error; err == nil && cfg.Value != "" {
			var protocols []string
			if json.Unmarshal([]byte(cfg.Value), &protocols) == nil {
				result[key] = protocols
				continue
			}
		}
		result[key] = defaultProtocolFilter[key]
	}
	utils.Success(c, result)
}

func AdminUpdateProtocolFilter(c *gin.Context) {
	var req map[string][]string
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	db := database.GetDB()
	for _, key := range []string{"clash_protocols", "universal_protocols"} {
		protocols, ok := req[key]
		if !ok {
			continue
		}
		val, _ := json.Marshal(protocols)
		result := db.Model(&models.SystemConfig{}).Where("category = ? AND `key` = ?", "protocol_filter", key).Updates(map[string]interface{}{"value": string(val)})
		if result.Error == nil && result.RowsAffected == 0 {
			db.Create(&models.SystemConfig{Category: "protocol_filter", Key: key, Value: string(val)})
		}
	}
	utils.CreateAuditLog(c, "update_protocol_filter", "settings", 0, "更新协议过滤设置")
	utils.InvalidateSettingsCache()
	utils.SuccessMessage(c, "协议过滤设置已保存")
}

func GetProtocolFilter(filterType string) map[string]bool {
	db := database.GetDB()
	var cfg models.SystemConfig
	if err := db.Where("category = ? AND `key` = ?", "protocol_filter", filterType).First(&cfg).Error; err != nil || cfg.Value == "" {
		return nil
	}
	var protocols []string
	if json.Unmarshal([]byte(cfg.Value), &protocols) != nil {
		return nil
	}
	m := make(map[string]bool, len(protocols))
	for _, p := range protocols {
		m[p] = true
	}
	return m
}

func FilterNodesByProtocol(nodes []models.Node, allowed map[string]bool) []models.Node {
	if allowed == nil {
		return nodes
	}
	var result []models.Node
	for _, n := range nodes {
		if allowed[n.Type] {
			result = append(result, n)
		}
	}
	return result
}

