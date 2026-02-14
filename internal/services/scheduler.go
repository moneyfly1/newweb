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
	var cfg models.SystemConfig
	siteName := "CBoard"
	if db.Where("`key` = ?", "site_name").First(&cfg).Error == nil && cfg.Value != "" {
		siteName = cfg.Value
	}

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
		QueueEmail(r.Email,
			fmt.Sprintf("%s - 订阅即将到期提醒", siteName),
			fmt.Sprintf(`<h3>订阅到期提醒</h3><p>您好，您的 %s 订阅将在 <strong>3 天</strong>后到期。</p><p>到期时间: %s</p><p>请及时续费以免服务中断。</p>`,
				siteName, r.ExpireTime.Format("2006-01-02 15:04")),
			"expiry_reminder")
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
		QueueEmail(r.Email,
			fmt.Sprintf("%s - 订阅明天到期", siteName),
			fmt.Sprintf(`<h3>订阅即将到期</h3><p>您好，您的 %s 订阅将在 <strong>1 天</strong>后到期。</p><p>到期时间: %s</p><p>请尽快续费！</p>`,
				siteName, r.ExpireTime.Format("2006-01-02 15:04")),
			"expiry_reminder")
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
		QueueEmail(r.Email,
			fmt.Sprintf("%s - 订阅已过期", siteName),
			fmt.Sprintf(`<h3>订阅已过期</h3><p>您好，您的 %s 订阅已于 %s 过期。</p><p>请续费以恢复服务。</p>`,
				siteName, r.ExpireTime.Format("2006-01-02 15:04")),
			"expiry_notice")
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