package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
)

// Scheduler manages all periodic background tasks.
type Scheduler struct {
	stopCh  chan struct{}
	wg      sync.WaitGroup
	mu      sync.Mutex
	running bool
}

var (
	scheduler     *Scheduler
	schedulerOnce sync.Once
)

// GetScheduler returns the singleton scheduler instance.
func GetScheduler() *Scheduler {
	schedulerOnce.Do(func() {
		scheduler = &Scheduler{stopCh: make(chan struct{})}
	})
	return scheduler
}

// Start launches all background task loops.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	log.Println("[Scheduler] 后台任务调度器已启动")
	utils.SysInfo("scheduler", "后台任务调度器已启动")

	s.startLoop("EmailQueue", 30*time.Second, processEmailQueueTask)
	s.startLoop("DeactivateExpired", 30*time.Minute, deactivateExpiredTask)
	s.startLoop("ExpiryCheck", 1*time.Hour, checkExpiryStatusTask)
	s.startLoop("ExpiryReminder", 6*time.Hour, sendExpiryRemindersTask)
	s.startLoop("UnpaidOrderReminder", 1*time.Hour, sendUnpaidOrderRemindersTask)
	s.startLoop("CleanCodes", 2*time.Hour, cleanExpiredCodesTask)
	s.startLoop("CancelExpiredOrders", 2*time.Hour, cancelExpiredOrdersTask)
	s.startLoop("CleanPaymentNonces", 12*time.Hour, cleanPaymentNoncesTask)
	s.startLoop("CleanExpiredTokens", 6*time.Hour, cleanExpiredTokensTask)
	s.startLoop("CleanOldLogs", 24*time.Hour, cleanOldLogsTask)
}

// Stop gracefully shuts down all background loops.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	close(s.stopCh)
	s.wg.Wait()
	s.running = false
	log.Println("[Scheduler] 后台任务调度器已停止")
}

func (s *Scheduler) startLoop(name string, interval time.Duration, fn func()) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		log.Printf("[Scheduler] 任务 %s 已启动 (间隔 %v)", name, interval)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		safeRun(name, fn)
		for {
			select {
			case <-ticker.C:
				safeRun(name, fn)
			case <-s.stopCh:
				log.Printf("[Scheduler] 任务 %s 已停止", name)
				return
			}
		}
	}()
}

func safeRun(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Scheduler] 任务 %s panic: %v", name, r)
			utils.SysError("scheduler", fmt.Sprintf("任务 %s panic", name), fmt.Sprintf("%v", r))
		}
	}()
	fn()
}

// ==================== Task Implementations ====================

// processEmailQueueTask processes pending emails in the queue.
func processEmailQueueTask() {
	ProcessEmailQueue()
}

// deactivateExpiredTask marks expired subscriptions as inactive.
func deactivateExpiredTask() {
	db := database.GetDB()
	result := db.Model(&models.Subscription{}).
		Where("is_active = ? AND expire_time < ?", true, time.Now()).
		Updates(map[string]interface{}{"is_active": false, "status": "expired"})
	if result.RowsAffected > 0 {
		log.Printf("[Scheduler] 已停用 %d 个过期订阅", result.RowsAffected)
		utils.SysInfo("scheduler", fmt.Sprintf("已停用 %d 个过期订阅", result.RowsAffected))
	}
}

// checkExpiryStatusTask marks subscriptions expiring within 24h.
func checkExpiryStatusTask() {
	db := database.GetDB()
	db.Model(&models.Subscription{}).
		Where("is_active = ? AND status = ? AND expire_time BETWEEN ? AND ?",
			true, "active", time.Now(), time.Now().Add(24*time.Hour)).
		Update("status", "expiring")
}

// sendExpiryRemindersTask sends reminder emails for expiring subscriptions.
func sendExpiryRemindersTask() {
	db := database.GetDB()

	type subUser struct {
		Email      string
		Username   string
		ExpireTime time.Time
	}

	// 3-day reminder (check a 1-hour window to avoid duplicates)
	now := time.Now()
	var remind3 []subUser
	db.Model(&models.Subscription{}).
		Select("users.email, users.username, subscriptions.expire_time").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Where("subscriptions.is_active = ? AND subscriptions.expire_time BETWEEN ? AND ? AND users.email_notifications = ?",
			true, now.Add(72*time.Hour), now.Add(73*time.Hour), true).
		Scan(&remind3)
	for _, r := range remind3 {
		subject, body := RenderEmail("expiry_reminder", map[string]string{
			"username": r.Username, "days": "3", "expire_time": r.ExpireTime.Format("2006-01-02 15:04"),
		})
		QueueEmail(r.Email, subject, body, "expiry_reminder")
		go NotifyAdmin("expiry_reminder", map[string]string{
			"username":    r.Username,
			"expire_time": r.ExpireTime.Format("2006-01-02 15:04"),
		})
	}

	// 1-day reminder
	var remind1 []subUser
	db.Model(&models.Subscription{}).
		Select("users.email, users.username, subscriptions.expire_time").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Where("subscriptions.is_active = ? AND subscriptions.expire_time BETWEEN ? AND ? AND users.email_notifications = ?",
			true, now.Add(24*time.Hour), now.Add(25*time.Hour), true).
		Scan(&remind1)
	for _, r := range remind1 {
		subject, body := RenderEmail("expiry_reminder", map[string]string{
			"username": r.Username, "days": "1", "expire_time": r.ExpireTime.Format("2006-01-02 15:04"),
		})
		QueueEmail(r.Email, subject, body, "expiry_reminder")
		go NotifyAdmin("expiry_reminder", map[string]string{
			"username":    r.Username,
			"expire_time": r.ExpireTime.Format("2006-01-02 15:04"),
		})
	}

	// Expired notice (within last hour)
	var expired []subUser
	db.Model(&models.Subscription{}).
		Select("users.email, subscriptions.expire_time").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Where("subscriptions.status = ? AND subscriptions.expire_time BETWEEN ? AND ? AND users.email_notifications = ?",
			"expired", now.Add(-1*time.Hour), now, true).
		Scan(&expired)
	for _, r := range expired {
		subject, body := RenderEmail("expiry_notice", map[string]string{
			"username": r.Username, "expire_time": r.ExpireTime.Format("2006-01-02 15:04"),
		})
		QueueEmail(r.Email, subject, body, "expiry_notice")
	}

	total := len(remind3) + len(remind1) + len(expired)
	if total > 0 {
		log.Printf("[Scheduler] 到期提醒: 3天=%d, 1天=%d, 已过期=%d", len(remind3), len(remind1), len(expired))
		utils.SysInfo("scheduler", fmt.Sprintf("到期提醒: 3天=%d, 1天=%d, 已过期=%d", len(remind3), len(remind1), len(expired)))
	}
}

// cleanExpiredCodesTask removes old verification codes.
func cleanExpiredCodesTask() {
	db := database.GetDB()
	result := db.Where("expires_at < ? OR used = ?", time.Now().Add(-24*time.Hour), 1).
		Delete(&models.VerificationCode{})
	if result.RowsAffected > 0 {
		log.Printf("[Scheduler] 已清理 %d 条过期验证码", result.RowsAffected)
		utils.SysInfo("scheduler", fmt.Sprintf("已清理 %d 条过期验证码", result.RowsAffected))
	}
}

// cancelExpiredOrdersTask cancels pending orders past their expire_time.
func cancelExpiredOrdersTask() {
	db := database.GetDB()
	result := db.Model(&models.Order{}).
		Where("status = ? AND expire_time < ?", "pending", time.Now()).
		Update("status", "expired")
	if result.RowsAffected > 0 {
		log.Printf("[Scheduler] 已取消 %d 个过期订单", result.RowsAffected)
		utils.SysInfo("scheduler", fmt.Sprintf("已取消 %d 个过期订单", result.RowsAffected))
	}
}

// cleanPaymentNoncesTask removes expired payment nonces.
func cleanPaymentNoncesTask() {
	db := database.GetDB()
	result := db.Where("expires_at < ?", time.Now()).Delete(&models.PaymentNonce{})
	if result.Error != nil {
		log.Printf("[Scheduler] 清理支付 nonce 失败: %v", result.Error)
		utils.SysError("scheduler", "清理支付 nonce 失败", result.Error.Error())
		return
	}
	if result.RowsAffected > 0 {
		log.Printf("[Scheduler] 已清理 %d 条过期支付 nonce", result.RowsAffected)
	}
}

// sendUnpaidOrderRemindersTask sends reminders for orders pending 15+ minutes.
func sendUnpaidOrderRemindersTask() {
	db := database.GetDB()
	now := time.Now()

	type orderUser struct {
		OrderNo     string
		UserID      uint
		Username    string
		Email       string
		Amount      float64
		FinalAmount *float64
		PackageID   uint
	}

	var orders []orderUser
	db.Model(&models.Order{}).
		Select("orders.order_no, orders.user_id, users.username, users.email, orders.amount, orders.final_amount, orders.package_id").
		Joins("JOIN users ON users.id = orders.user_id").
		Where("orders.status = ? AND orders.created_at BETWEEN ? AND ? AND users.email_notifications = ?",
			"pending", now.Add(-20*time.Minute), now.Add(-15*time.Minute), true).
		Scan(&orders)

	// 批量加载所有涉及的套餐名称，避免循环内逐条查询
	pkgMap := make(map[uint]string)
	if len(orders) > 0 {
		var pkgIDs []uint
		for _, o := range orders {
			pkgIDs = append(pkgIDs, o.PackageID)
		}
		var pkgs []models.Package
		db.Where("id IN ?", pkgIDs).Select("id, name").Find(&pkgs)
		for _, p := range pkgs {
			pkgMap[p.ID] = p.Name
		}
	}

	for _, o := range orders {
		amount := o.Amount
		if o.FinalAmount != nil {
			amount = *o.FinalAmount
		}
		pkgName := pkgMap[o.PackageID]
		if pkgName == "" {
			pkgName = "未知套餐"
		}
		subject, body := RenderEmail("unpaid_order", map[string]string{
			"username": o.Username, "order_no": o.OrderNo, "package_name": pkgName,
			"amount": fmt.Sprintf("%.2f", amount),
		})
		QueueEmail(o.Email, subject, body, "unpaid_order")
		go NotifyAdmin("unpaid_order", map[string]string{
			"order_no": o.OrderNo,
			"username": o.Username,
			"amount":   fmt.Sprintf("%.2f", amount),
		})
	}

	if len(orders) > 0 {
		log.Printf("[Scheduler] 未付款订单提醒: %d 条", len(orders))
		utils.SysInfo("scheduler", fmt.Sprintf("未付款订单提醒: %d 条", len(orders)))
	}
}

// cleanExpiredTokensTask removes expired JWT tokens from the blacklist.
func cleanExpiredTokensTask() {
	db := database.GetDB()
	result := db.Where("expires_at < ?", time.Now()).Delete(&models.TokenBlacklist{})
	if result.RowsAffected > 0 {
		log.Printf("[Scheduler] 已清理 %d 条过期令牌", result.RowsAffected)
	}
}

// cleanOldLogsTask removes log records older than the configured retention period.
func cleanOldLogsTask() {
	retentionDays := utils.GetIntSetting("log_retention_days", 90)
	if retentionDays <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	db := database.GetDB()
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
		result := db.Where("created_at < ?", cutoff).Delete(t.model)
		if result.Error != nil {
			log.Printf("[Scheduler] 清理 %s 失败: %v", t.name, result.Error)
			continue
		}
		if result.RowsAffected > 0 {
			totalDeleted += result.RowsAffected
		}
	}

	// Also clean old file-based logs
	cleanOldLogFiles(retentionDays)

	if totalDeleted > 0 {
		log.Printf("[Scheduler] 日志清理完成: 共删除 %d 条 %d 天前的记录", totalDeleted, retentionDays)
		utils.SysInfo("scheduler", fmt.Sprintf("日志清理完成: 共删除 %d 条 %d 天前的记录", totalDeleted, retentionDays))
	}
}

// cleanOldLogFiles removes app log files older than the given number of days.
func cleanOldLogFiles(retentionDays int) {
	logDir := "logs"
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Match app-YYYY-MM-DD.log pattern
		if !strings.HasPrefix(name, "app-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		dateStr := strings.TrimSuffix(strings.TrimPrefix(name, "app-"), ".log")
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if fileDate.Before(cutoff) {
			filePath := filepath.Join(logDir, name)
			if err := os.Remove(filePath); err == nil {
				deleted++
			}
		}
	}
	if deleted > 0 {
		log.Printf("[Scheduler] 已清理 %d 个过期日志文件", deleted)
	}
}
