package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
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

	s.startLoop("EmailQueue", 10*time.Second, processEmailQueueTask)
	s.startLoop("DeactivateExpired", 5*time.Minute, deactivateExpiredTask)
	s.startLoop("ExpiryCheck", 10*time.Minute, checkExpiryStatusTask)
	s.startLoop("ExpiryReminder", 1*time.Hour, sendExpiryRemindersTask)
	s.startLoop("UnpaidOrderReminder", 15*time.Minute, sendUnpaidOrderRemindersTask)
	s.startLoop("CleanCodes", 30*time.Minute, cleanExpiredCodesTask)
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
		ExpireTime time.Time
	}

	// 3-day reminder (check a 1-hour window to avoid duplicates)
	var remind3 []subUser
	db.Model(&models.Subscription{}).
		Select("users.email, subscriptions.expire_time").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Where("subscriptions.is_active = ? AND subscriptions.expire_time BETWEEN ? AND ? AND users.email_notifications = ?",
			true, time.Now().Add(72*time.Hour), time.Now().Add(73*time.Hour), true).
		Scan(&remind3)
	for _, r := range remind3 {
		subject, body := RenderEmail("expiry_reminder", map[string]string{
			"days": "3", "expire_time": r.ExpireTime.Format("2006-01-02 15:04"),
		})
		QueueEmail(r.Email, subject, body, "expiry_reminder")
	}

	// 1-day reminder
	var remind1 []subUser
	db.Model(&models.Subscription{}).
		Select("users.email, subscriptions.expire_time").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Where("subscriptions.is_active = ? AND subscriptions.expire_time BETWEEN ? AND ? AND users.email_notifications = ?",
			true, time.Now().Add(24*time.Hour), time.Now().Add(25*time.Hour), true).
		Scan(&remind1)
	for _, r := range remind1 {
		subject, body := RenderEmail("expiry_reminder", map[string]string{
			"days": "1", "expire_time": r.ExpireTime.Format("2006-01-02 15:04"),
		})
		QueueEmail(r.Email, subject, body, "expiry_reminder")
	}

	// Expired notice (within last hour)
	var expired []subUser
	db.Model(&models.Subscription{}).
		Select("users.email, subscriptions.expire_time").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Where("subscriptions.status = ? AND subscriptions.expire_time BETWEEN ? AND ? AND users.email_notifications = ?",
			"expired", time.Now().Add(-1*time.Hour), time.Now(), true).
		Scan(&expired)
	for _, r := range expired {
		subject, body := RenderEmail("expiry_notice", map[string]string{
			"expire_time": r.ExpireTime.Format("2006-01-02 15:04"),
		})
		QueueEmail(r.Email, subject, body, "expiry_notice")
	}

	total := len(remind3) + len(remind1) + len(expired)
	if total > 0 {
		log.Printf("[Scheduler] 到期提醒: 3天=%d, 1天=%d, 已过期=%d", len(remind3), len(remind1), len(expired))
	}
}

// cleanExpiredCodesTask removes old verification codes.
func cleanExpiredCodesTask() {
	db := database.GetDB()
	result := db.Where("expires_at < ? OR used = ?", time.Now().Add(-24*time.Hour), 1).
		Delete(&models.VerificationCode{})
	if result.RowsAffected > 0 {
		log.Printf("[Scheduler] 已清理 %d 条过期验证码", result.RowsAffected)
	}
}

// sendUnpaidOrderRemindersTask sends reminders for orders pending 15+ minutes.
func sendUnpaidOrderRemindersTask() {
	db := database.GetDB()

	type orderUser struct {
		OrderNo     string
		UserID      uint
		Email       string
		Amount      float64
		FinalAmount *float64
		PackageID   uint
	}

	var orders []orderUser
	db.Model(&models.Order{}).
		Select("orders.order_no, orders.user_id, users.email, orders.amount, orders.final_amount, orders.package_id").
		Joins("JOIN users ON users.id = orders.user_id").
		Where("orders.status = ? AND orders.created_at BETWEEN ? AND ? AND users.email_notifications = ?",
			"pending", time.Now().Add(-20*time.Minute), time.Now().Add(-15*time.Minute), true).
		Scan(&orders)

	for _, o := range orders {
		amount := o.Amount
		if o.FinalAmount != nil {
			amount = *o.FinalAmount
		}
		var pkgName string
		db.Model(&models.Package{}).Where("id = ?", o.PackageID).Pluck("name", &pkgName)
		if pkgName == "" {
			pkgName = "未知套餐"
		}
		subject, body := RenderEmail("unpaid_order", map[string]string{
			"order_no": o.OrderNo, "package_name": pkgName,
			"amount": fmt.Sprintf("%.2f", amount),
		})
		QueueEmail(o.Email, subject, body, "unpaid_order")
	}

	if len(orders) > 0 {
		log.Printf("[Scheduler] 未付款订单提醒: %d 条", len(orders))
	}
}