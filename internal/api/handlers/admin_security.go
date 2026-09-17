package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/api/middleware"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 后台「登录限制 / 解封」。
//
// 背景：客户连续输错密码后可能被限制，而限制有两套完全不同的机制，
// 客户看到的提示也不一样（「登录失败次数过多」/「请求过于频繁」），
// 后台此前没有任何地方能看到「谁被限了、还剩多久、怎么解」：
//  1. 账号+IP 锁定（数据库 login_attempts）：同一邮箱同一 IP 在窗口内失败超阈值即锁定；
//  2. IP 接口限流（Redis 或进程内存）：某 IP 对某接口每分钟请求超上限即限流，
//     /auth/login 是 10 次/分钟，/subscription/reset 是 3 次/30 分钟（客户最容易踩的就是这个）。
// 本文件提供：列出当前生效的限制、按账号/IP 立即解封（同时清数据库记录与限流计数），
// 以及把锁定配置的当前状态一并回显，避免「以为开着锁定其实阈值是 2000」这种误判。

// LoginLimitStatus 后台页面用的限制总览
type LoginLimitStatus struct {
	MaxAttempts    int                         `json:"max_login_attempts"`
	LockoutMinutes int                         `json:"login_lockout_minutes"`
	LockoutEnabled bool                        `json:"lockout_enabled"`
	LockoutNote    string                      `json:"lockout_note"`
	LockedAccounts []LockedAccount             `json:"locked_accounts"`
	LockedVerify   []LockedVerification        `json:"locked_verifications"`
	RateLimited    []middleware.RateLimitEntry `json:"rate_limited"`
	RecentFailures []RecentFailure             `json:"recent_failures"`
	RedisEnabled   bool                        `json:"redis_enabled"`
}

// LockedAccount 被「账号+IP」锁定的记录
type LockedAccount struct {
	Username        string    `json:"username"`
	IPAddress       string    `json:"ip_address"`
	FailCount       int64     `json:"fail_count"`
	FirstAttempt    time.Time `json:"first_attempt"`
	LastAttempt     time.Time `json:"last_attempt"`
	UnlockAt        time.Time `json:"unlock_at"`
	RemainingSecond int64     `json:"remaining_seconds"`
}

// LockedVerification 被锁定的邮箱验证码流程（注册/找回密码）
type LockedVerification struct {
	Email           string    `json:"email"`
	Purpose         string    `json:"purpose"`
	FailCount       int64     `json:"fail_count"`
	LastAttempt     time.Time `json:"last_attempt"`
	RemainingSecond int64     `json:"remaining_seconds"`
}

// RecentFailure 近 24 小时失败次数最多的账号（客服排查用）
type RecentFailure struct {
	Username    string    `json:"username"`
	IPAddress   string    `json:"ip_address"`
	FailCount   int64     `json:"fail_count"`
	LastAttempt time.Time `json:"last_attempt"`
}

// loginLockoutConfig 读取登录锁定配置。
// 约定：max_login_attempts <= 0 或 login_lockout_minutes <= 0 都表示「不锁定」
// （线上就出现过 2000 / 0 这种等于关闭的配置，必须显式告诉管理员）。
func loginLockoutConfig() (maxAttempts int, lockoutMinutes int, enabled bool) {
	maxAttempts = utils.GetIntSetting("max_login_attempts", 5)
	lockoutMinutes = utils.GetIntSetting("login_lockout_minutes", 30)
	enabled = maxAttempts > 0 && lockoutMinutes > 0
	return
}

// AdminListLoginLimits 列出当前所有生效的登录限制（账号锁定 + 邮箱验证锁定 + IP 限流）
func AdminListLoginLimits(c *gin.Context) {
	db := database.GetDB()
	maxAttempts, lockoutMinutes, enabled := loginLockoutConfig()

	status := LoginLimitStatus{
		MaxAttempts:    maxAttempts,
		LockoutMinutes: lockoutMinutes,
		LockoutEnabled: enabled,
		LockedAccounts: []LockedAccount{},
		LockedVerify:   []LockedVerification{},
		RateLimited:    []middleware.RateLimitEntry{},
		RecentFailures: []RecentFailure{},
		RedisEnabled:   database.GetRedis() != nil,
	}
	switch {
	case !enabled:
		status.LockoutNote = fmt.Sprintf("当前未启用账号锁定（最大失败次数 %d、锁定时长 %d 分钟，任一项为 0 即视为关闭）", maxAttempts, lockoutMinutes)
	default:
		status.LockoutNote = fmt.Sprintf("同一账号在同一 IP 上 %d 分钟内失败 %d 次即锁定 %d 分钟", lockoutMinutes, maxAttempts, lockoutMinutes)
	}

	// 1) 账号 + IP 锁定
	if enabled {
		since := time.Now().Add(-time.Duration(lockoutMinutes) * time.Minute)
		// 时间列必须用 flexTime：SQLite 驱动把 datetime 返回成字符串，
		// 直接扫进 time.Time 会静默留成零值（0001-01-01），
		// 界面上就会显示成「剩余 0 秒 / 1970 年」这类明显不对的信息。
		type row struct {
			Username     string
			IPAddress    *string
			FailCount    int64
			FirstAttempt flexTime
			LastAttempt  flexTime
		}
		var rows []row
		db.Model(&models.LoginAttempt{}).
			Select("username, ip_address, COUNT(*) as fail_count, MIN(created_at) as first_attempt, MAX(created_at) as last_attempt").
			Where("success = 0 AND created_at > ?", since).
			Group("username, ip_address").
			Having("COUNT(*) >= ?", maxAttempts).
			Order("last_attempt DESC").
			Scan(&rows)

		for _, r := range rows {
			ip := ""
			if r.IPAddress != nil {
				ip = *r.IPAddress
			}
			first := r.FirstAttempt.Time
			unlockAt := first.Add(time.Duration(lockoutMinutes) * time.Minute)
			status.LockedAccounts = append(status.LockedAccounts, LockedAccount{
				Username:        r.Username,
				IPAddress:       ip,
				FailCount:       r.FailCount,
				FirstAttempt:    first,
				LastAttempt:     r.LastAttempt.Time,
				UnlockAt:        unlockAt,
				RemainingSecond: maxInt64(0, int64(time.Until(unlockAt).Seconds())),
			})
		}
	}

	// 2) 邮箱验证码锁定（注册 / 找回密码）
	if enabled {
		since := time.Now().Add(-time.Duration(lockoutMinutes) * time.Minute)
		type vrow struct {
			Email       string
			Purpose     string
			FailCount   int64
			LastAttempt flexTime
		}
		var vrows []vrow
		db.Model(&models.VerificationAttempt{}).
			Select("email, purpose, COUNT(*) as fail_count, MAX(created_at) as last_attempt").
			Where("success = 0 AND created_at > ?", since).
			Group("email, purpose").
			Having("COUNT(*) >= ?", maxAttempts).
			Order("last_attempt DESC").
			Scan(&vrows)
		for _, r := range vrows {
			status.LockedVerify = append(status.LockedVerify, LockedVerification{
				Email:       r.Email,
				Purpose:     r.Purpose,
				FailCount:   r.FailCount,
				LastAttempt: r.LastAttempt.Time,
			})
		}
	}

	// 3) IP 接口限流（Redis + 进程内存两侧都列）
	status.RateLimited = append(status.RateLimited, middleware.ListRedisRateLimits()...)
	status.RateLimited = append(status.RateLimited, middleware.ListMemoryLimiters()...)

	// 4) 近 24 小时失败最多的账号：客户来说「登不上」时先看这里
	since24 := time.Now().Add(-24 * time.Hour)
	type frow struct {
		Username    string
		IPAddress   *string
		FailCount   int64
		LastAttempt flexTime
	}
	var frows []frow
	db.Model(&models.LoginAttempt{}).
		Select("username, ip_address, COUNT(*) as fail_count, MAX(created_at) as last_attempt").
		Where("success = 0 AND created_at > ?", since24).
		Group("username, ip_address").
		Order("fail_count DESC, last_attempt DESC").
		Limit(20).
		Scan(&frows)
	for _, r := range frows {
		ip := ""
		if r.IPAddress != nil {
			ip = *r.IPAddress
		}
		status.RecentFailures = append(status.RecentFailures, RecentFailure{
			Username: r.Username, IPAddress: ip, FailCount: r.FailCount, LastAttempt: r.LastAttempt.Time,
		})
	}

	utils.Success(c, status)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// AdminUnlockLogin 解封：按用户 / 账号 / IP 清除限制。
//
// 三件事一起做，避免「解了账号还限着 IP」：
//   - 删除该账号（及其 IP）的登录失败记录 → 账号不再被锁
//   - 删除该邮箱的验证码失败记录 → 找回密码/注册流程不再被锁
//   - 清除该 IP 的接口限流计数（Redis + 内存）→ 立即可以再请求
func AdminUnlockLogin(c *gin.Context) {
	var req struct {
		UserID     uint   `json:"user_id"`
		Identifier string `json:"identifier"` // 邮箱或用户名
		IPAddress  string `json:"ip_address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	identifiers := []string{}

	// 1) 按用户 ID：取该用户的邮箱与用户名（登录尝试按「用户输入的那串」记录，两种都要清）
	var targetUser models.User
	if req.UserID > 0 {
		if err := db.First(&targetUser, req.UserID).Error; err != nil {
			utils.NotFound(c, "用户不存在")
			return
		}
		if targetUser.Email != "" {
			identifiers = append(identifiers, strings.ToLower(targetUser.Email))
		}
		if targetUser.Username != "" {
			identifiers = append(identifiers, strings.ToLower(targetUser.Username))
		}
	}
	if id := strings.TrimSpace(strings.ToLower(req.Identifier)); id != "" {
		identifiers = append(identifiers, id)
		// 允许管理员直接填邮箱或用户名
		var u models.User
		if err := db.Where("LOWER(email) = ? OR LOWER(username) = ?", id, id).First(&u).Error; err == nil {
			if u.Email != "" {
				identifiers = append(identifiers, strings.ToLower(u.Email))
			}
			if u.Username != "" {
				identifiers = append(identifiers, strings.ToLower(u.Username))
			}
			targetUser = u
		}
	}

	ip := strings.TrimSpace(req.IPAddress)
	if len(identifiers) == 0 && ip == "" {
		utils.BadRequest(c, "请提供 user_id、identifier（邮箱/用户名）或 ip_address 之一")
		return
	}

	unique := map[string]struct{}{}
	list := make([]string, 0, len(identifiers))
	for _, v := range identifiers {
		if v == "" {
			continue
		}
		if _, ok := unique[v]; ok {
			continue
		}
		unique[v] = struct{}{}
		list = append(list, v)
	}

	deletedAttempts := int64(0)
	deletedVerify := int64(0)

	err := db.Transaction(func(tx *gorm.DB) error {
		if len(list) > 0 {
			// 只删这个账号自己的失败记录（保留其它账号/IP 的失败证据）
			r := tx.Where("success = 0 AND LOWER(username) IN ?", list).Delete(&models.LoginAttempt{})
			if r.Error != nil {
				return r.Error
			}
			deletedAttempts += r.RowsAffected
			rv := tx.Where("success = 0 AND LOWER(email) IN ?", list).Delete(&models.VerificationAttempt{})
			if rv.Error != nil {
				return rv.Error
			}
			deletedVerify += rv.RowsAffected
		}
		if ip != "" {
			r := tx.Where("success = 0 AND ip_address = ?", ip).Delete(&models.LoginAttempt{})
			if r.Error != nil {
				return r.Error
			}
			deletedAttempts += r.RowsAffected
		}
		return nil
	})
	if err != nil {
		utils.InternalError(c, "解封失败: "+err.Error())
		return
	}

	// 清除接口限流计数（Redis + 内存）
	clearedRedis := 0
	clearedMemory := 0
	if ip != "" {
		clearedRedis = middleware.ClearRedisRateLimits(ip)
		clearedMemory = middleware.ResetMemoryLimiters(ip)
	} else {
		// 只给了账号：把该账号近期失败记录里出现过的 IP 一并放行，
		// 否则客户换了网络还是会在同一分钟内被限流
		ips := recentFailedIPs(db, list)
		for _, one := range ips {
			clearedRedis += middleware.ClearRedisRateLimits(one)
			clearedMemory += middleware.ResetMemoryLimiters(one)
		}
	}

	target := strings.Join(list, " / ")
	if target == "" {
		target = ip
	}
	if ip != "" {
		target += " (" + ip + ")"
	}
	utils.CreateAuditLog(c, "unlock_login_limit", "user", targetUser.ID, fmt.Sprintf("解除登录限制: %s, 删除失败记录 %d 条, 清除限流计数 redis=%d memory=%d", target, deletedAttempts+deletedVerify, clearedRedis, clearedMemory))

	utils.Success(c, gin.H{
		"identifiers":            list,
		"ip_address":             ip,
		"deleted_login_attempts": deletedAttempts,
		"deleted_verifications":  deletedVerify,
		"cleared_redis_keys":     clearedRedis,
		"cleared_memory_entries": clearedMemory,
		"message":                "已解除限制，客户现在可以立即重试",
	})
}

// recentFailedIPs 取这些账号近期失败记录里出现过的 IP（去重，最多 10 个）
func recentFailedIPs(db *gorm.DB, identifiers []string) []string {
	if len(identifiers) == 0 {
		return nil
	}
	var ips []string
	db.Model(&models.LoginAttempt{}).
		Where("success = 0 AND LOWER(username) IN ? AND ip_address IS NOT NULL AND ip_address != ''", identifiers).
		Distinct().
		Order("created_at DESC").
		Limit(10).
		Pluck("ip_address", &ips)
	return ips
}

// AdminClearAllRateLimits 清空全部 IP 限流计数（应急用：线上大面积误限流时一把清掉）
func AdminClearAllRateLimits(c *gin.Context) {
	deleted := middleware.ClearRedisRateLimits("")
	cleared := middleware.ResetMemoryLimiters("")
	utils.CreateAuditLog(c, "clear_all_rate_limits", "settings", 0, fmt.Sprintf("清空全部 IP 限流计数: redis=%d memory=%d", deleted, cleared))
	utils.Success(c, gin.H{"cleared_redis_keys": deleted, "cleared_memory_entries": cleared})
}

// AdminUnlockSuggestion 供前端在用户详情里判断「这个用户当前是否被限制」。
// 返回该用户所有标识下的失败次数与最早失败时间，不修改任何数据。
func AdminUnlockSuggestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, uint(id)).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}
	identifiers := []string{}
	if user.Email != "" {
		identifiers = append(identifiers, strings.ToLower(user.Email))
	}
	if user.Username != "" {
		identifiers = append(identifiers, strings.ToLower(user.Username))
	}

	maxAttempts, lockoutMinutes, enabled := loginLockoutConfig()
	since := time.Now().Add(-time.Duration(lockoutMinutes) * time.Minute)
	if !enabled {
		since = time.Now().Add(-24 * time.Hour)
	}

	var fails int64
	var earliest, latest *time.Time
	if len(identifiers) > 0 {
		db.Model(&models.LoginAttempt{}).
			Where("success = 0 AND LOWER(username) IN ? AND created_at > ?", identifiers, since).
			Count(&fails)
		type trow struct {
			First *flexTime
			Last  *flexTime
		}
		var tr trow
		db.Model(&models.LoginAttempt{}).
			Select("MIN(created_at) as first, MAX(created_at) as last").
			Where("success = 0 AND LOWER(username) IN ? AND created_at > ?", identifiers, since).
			Scan(&tr)
		if tr.First != nil {
			t := tr.First.Time
			earliest = &t
		}
		if tr.Last != nil {
			t := tr.Last.Time
			latest = &t
		}
	}

	limited := enabled && fails >= int64(maxAttempts)
	utils.Success(c, gin.H{
		"user_id":            user.ID,
		"identifiers":        identifiers,
		"fail_count":         fails,
		"window_minutes":     lockoutMinutes,
		"max_login_attempts": maxAttempts,
		"lockout_enabled":    enabled,
		"limited":            limited,
		"first_attempt_at":   earliest,
		"last_attempt_at":    latest,
	})
}
