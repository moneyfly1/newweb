package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"cboard/v2/internal/api/middleware"
	"cboard/v2/internal/config"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Register 用户注册
func Register(c *gin.Context) {
	var req struct {
		Username         string `json:"username" binding:"required,min=3,max=50"`
		Email            string `json:"email" binding:"required,email"`
		Password         string `json:"password" binding:"required,min=6"`
		InviteCode       string `json:"invite_code"`
		VerificationCode string `json:"verification_code"`
		Honeypot         string `json:"website"` // 蜜罐字段，正常用户不应填写
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 蜜罐检测：如果填写了 website 字段，说明是机器人
	if req.Honeypot != "" {
		utils.SysError("security", fmt.Sprintf("注册蜜罐触发: email=%s, honeypot=%s", req.Email, req.Honeypot))
		// 返回成功但不实际创建账户，迷惑机器人
		utils.Success(c, gin.H{
			"user":          gin.H{"id": 0, "username": req.Username, "email": req.Email},
			"access_token":  "fake_token",
			"refresh_token": "fake_token",
		})
		return
	}

	// 用户名/邮箱格式验证
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// 防止 SQL 注入特殊字符（虽然 GORM 已参数化，但额外检查）
	if strings.ContainsAny(req.Username, "'\"\\<>") {
		utils.BadRequest(c, "用户名包含非法字符")
		return
	}

	db := database.GetDB()

	// 读取注册相关设置
	settings := utils.GetSettings(
		"register_enabled", "register_email_verify", "register_invite_required",
		"default_device_limit", "default_subscribe_days", "min_password_length",
	)

	// 检查是否允许注册
	regEnabled := settings["register_enabled"]
	if regEnabled == "false" || regEnabled == "0" {
		utils.Forbidden(c, "暂未开放注册")
		return
	}

	// 检查密码最小长度
	minPwdLen := 6
	if v := settings["min_password_length"]; v != "" {
		if n, err := fmt.Sscanf(v, "%d", &minPwdLen); n == 0 || err != nil {
			minPwdLen = 6
		}
	}
	if len(req.Password) < minPwdLen {
		utils.BadRequest(c, fmt.Sprintf("密码长度不能少于 %d 位", minPwdLen))
		return
	}
	if err := utils.ValidatePasswordStrength(req.Password); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 检查是否需要邀请码
	inviteRequired := settings["register_invite_required"] == "true" || settings["register_invite_required"] == "1"
	if inviteRequired && req.InviteCode == "" {
		utils.BadRequest(c, "注册需要邀请码")
		return
	}

	// 检查是否需要邮箱验证
	emailVerify := settings["register_email_verify"] == "true" || settings["register_email_verify"] == "1"
	if emailVerify {
		if req.VerificationCode == "" {
			utils.BadRequest(c, "请先完成邮箱验证")
			return
		}
		var vc models.VerificationCode
		if err := db.Where("email = ? AND code = ? AND purpose = ? AND used = 0 AND expires_at > ?",
			req.Email, req.VerificationCode, "register", time.Now()).Order("created_at DESC").First(&vc).Error; err != nil {
			utils.BadRequest(c, "验证码无效或已过期")
			return
		}
		vc.MarkAsUsed()
		if err := db.Save(&vc).Error; err != nil {
			utils.InternalError(c, "更新验证码状态失败")
			return
		}
	}

	// 检查用户名和邮箱是否已存在
	var count int64
	db.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		utils.Conflict(c, "用户名已存在")
		return
	}
	db.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		utils.Conflict(c, "邮箱已被注册")
		return
	}

	// 密码加密
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.InternalError(c, "密码加密失败")
		return
	}

	user := models.User{
		Username:                    req.Username,
		Email:                       req.Email,
		Password:                    hashedPassword,
		IsActive:                    true,
		Theme:                       "light",
		Language:                    "zh-CN",
		Timezone:                    "Asia/Shanghai",
		EmailNotifications:          true,
		AbnormalLoginAlertEnabled:   true,
		PushNotifications:           true,
		DataSharing:                 true,
		Analytics:                   true,
		SpecialNodeSubscriptionType: "both",
	}

	// 处理邀请码
	if req.InviteCode != "" {
		// Validate invite code existence only (detailed check with locking inside transaction)
		var inviteCheck models.InviteCode
		if err := db.Where("UPPER(code) = UPPER(?) AND is_active = ?", req.InviteCode, true).First(&inviteCheck).Error; err != nil {
			utils.BadRequest(c, "邀请码无效或已失效")
			return
		}
		invitedBy := inviteCheck.UserID
		user.InvitedBy = &invitedBy
		user.InviteCodeUsed = &req.InviteCode
	}

	// 开启事务：创建用户、更新邀请码、发放奖励、创建订阅
	tx := db.Begin()
	if tx.Error != nil {
		utils.InternalError(c, "创建事务失败")
		return
	}

	// Re-validate invite code with row lock inside transaction
	if req.InviteCode != "" {
		var inviteCode models.InviteCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("UPPER(code) = UPPER(?) AND is_active = ?", req.InviteCode, true).First(&inviteCode).Error; err != nil {
			tx.Rollback()
			utils.BadRequest(c, "邀请码无效或已失效")
			return
		}
		if inviteCode.MaxUses != nil && inviteCode.UsedCount >= int(*inviteCode.MaxUses) {
			tx.Rollback()
			utils.BadRequest(c, "邀请码已达到最大使用次数")
			return
		}
		if inviteCode.ExpiresAt != nil && inviteCode.ExpiresAt.Before(time.Now()) {
			tx.Rollback()
			utils.BadRequest(c, "邀请码已过期")
			return
		}
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "创建用户失败")
		return
	}

	// 更新邀请码使用次数
	if req.InviteCode != "" {
		if err := tx.Model(&models.InviteCode{}).Where("UPPER(code) = UPPER(?)", req.InviteCode).
			UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
			tx.Rollback()
			utils.InternalError(c, "更新邀请码失败")
			return
		}
	}

	// 创建邀请关系 + 发放奖励
	if req.InviteCode != "" && user.InvitedBy != nil {
		var inviteCode models.InviteCode
		if err := tx.Where("UPPER(code) = UPPER(?)", req.InviteCode).First(&inviteCode).Error; err == nil {
			relation := models.InviteRelation{
				InviteCodeID: inviteCode.ID,
				InviterID:    inviteCode.UserID,
				InviteeID:    user.ID,
			}
			// 发放邀请人奖励
			if inviteCode.InviterReward > 0 {
				relation.InviterRewardAmount = inviteCode.InviterReward
				relation.InviterRewardGiven = true
				var inviter models.User
				if tx.First(&inviter, inviteCode.UserID).Error == nil {
					if err := tx.Model(&inviter).UpdateColumn("balance", gorm.Expr("balance + ?", inviteCode.InviterReward)).Error; err != nil {
						tx.Rollback()
						utils.InternalError(c, "发放邀请奖励失败")
						return
					}
					desc := fmt.Sprintf("邀请用户 %s 注册奖励", user.Username)
					if err := tx.Create(&models.BalanceLog{
						UserID:        inviter.ID,
						ChangeType:    "invite_reward",
						Amount:        inviteCode.InviterReward,
						BalanceBefore: inviter.Balance,
						BalanceAfter:  inviter.Balance + inviteCode.InviterReward,
						Description:   &desc,
					}).Error; err != nil {
						tx.Rollback()
						utils.InternalError(c, "记录邀请奖励失败")
						return
					}
				}
			}
			// 发放被邀请人奖励
			if inviteCode.InviteeReward > 0 {
				relation.InviteeRewardAmount = inviteCode.InviteeReward
				relation.InviteeRewardGiven = true
				if err := tx.Model(&user).UpdateColumn("balance", gorm.Expr("balance + ?", inviteCode.InviteeReward)).Error; err != nil {
					tx.Rollback()
					utils.InternalError(c, "发放受邀奖励失败")
					return
				}
				desc := fmt.Sprintf("受邀注册奖励 (邀请码: %s)", req.InviteCode)
				if err := tx.Create(&models.BalanceLog{
					UserID:        user.ID,
					ChangeType:    "invite_reward",
					Amount:        inviteCode.InviteeReward,
					BalanceBefore: 0,
					BalanceAfter:  inviteCode.InviteeReward,
					Description:   &desc,
				}).Error; err != nil {
					tx.Rollback()
					utils.InternalError(c, "记录受邀奖励失败")
					return
				}
			}
			if err := tx.Create(&relation).Error; err != nil {
				tx.Rollback()
				utils.InternalError(c, "创建邀请关系失败")
				return
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			utils.InternalError(c, "读取邀请码失败")
			return
		}
	}

	// Auto-create subscription for new user
	deviceLimit := 3
	if v := settings["default_device_limit"]; v != "" {
		if n, err := fmt.Sscanf(v, "%d", &deviceLimit); n == 0 || err != nil {
			deviceLimit = 3
		}
	}
	subscribeDays := 0
	if v := settings["default_subscribe_days"]; v != "" {
		if _, err := fmt.Sscanf(v, "%d", &subscribeDays); err != nil {
			subscribeDays = 0
		}
	}
	expireTime := time.Now()
	if subscribeDays > 0 {
		expireTime = time.Now().AddDate(0, 0, subscribeDays)
	}

	subURL := utils.GenerateHexToken()
	subscription := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: subURL,
		DeviceLimit:     deviceLimit,
		IsActive:        subscribeDays > 0,
		Status:          "active",
		ExpireTime:      expireTime,
	}
	if err := tx.Create(&subscription).Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "创建订阅失败")
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.InternalError(c, "注册事务提交失败")
		return
	}

	// 记录注册日志
	var inviterIDPtr *uint
	if user.InvitedBy != nil {
		inviterIDPtr = new(uint)
		*inviterIDPtr = uint(*user.InvitedBy)
	}
	utils.CreateRegistrationLog(c, user.ID, user.Username, user.Email, req.InviteCode, inviterIDPtr)

	// 发送欢迎邮件 + 通知管理员
	welcomeSubject, welcomeBody := services.RenderEmail("welcome", map[string]string{
		"username": user.Username,
		"email":    user.Email,
	})
	go services.QueueEmail(user.Email, welcomeSubject, welcomeBody, "welcome")
	go services.NotifyAdmin("new_user", map[string]string{"username": user.Username, "email": user.Email})

	// 生成 Token
	accessToken, err := generateToken(user.ID, "access", time.Duration(config.AppConfig.AccessTokenExpireMinutes)*time.Minute)
	if err != nil {
		utils.InternalError(c, "生成访问令牌失败")
		return
	}
	refreshToken, err := generateToken(user.ID, "refresh", time.Duration(config.AppConfig.RefreshTokenExpireDays)*24*time.Hour)
	if err != nil {
		utils.InternalError(c, "生成刷新令牌失败")
		return
	}

	utils.Success(c, gin.H{
		"user":          gin.H{"id": user.ID, "username": user.Username, "email": user.Email},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Login 用户登录
func Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	db := database.GetDB()
	clientIP := utils.GetRealClientIP(c)
	userAgent := c.GetHeader("User-Agent")

	// 检查登录锁定（基于IP+用户名）
	maxAttempts := utils.GetIntSetting("max_login_attempts", 5)
	lockoutMinutes := utils.GetIntSetting("login_lockout_minutes", 30)
	if maxAttempts > 0 {
		var failCount int64
		since := time.Now().Add(-time.Duration(lockoutMinutes) * time.Minute)
		db.Model(&models.LoginAttempt{}).
			Where("username = ? AND success = 0 AND created_at > ?", req.Email, since).
			Count(&failCount)
		if failCount >= int64(maxAttempts) {
			// 查询最早的失败记录，计算准确的解锁剩余时间
			var earliest models.LoginAttempt
			db.Where("username = ? AND success = 0 AND created_at > ?", req.Email, since).
				Order("created_at ASC").First(&earliest)
			unlockAt := earliest.CreatedAt.Add(time.Duration(lockoutMinutes) * time.Minute)
			retryAfterSec := int(time.Until(unlockAt).Seconds())
			if retryAfterSec < 0 {
				retryAfterSec = 0
			}
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSec))
			utils.TooManyRequests(c, fmt.Sprintf("登录失败次数过多，请 %d 分钟后再试", lockoutMinutes))
			return
		}
	}

	// 记录登录尝试
	attempt := models.LoginAttempt{
		Username:  req.Email,
		IPAddress: &clientIP,
		UserAgent: &userAgent,
	}

	var user models.User
	userNotFound := db.Where("email = ? OR username = ?", req.Email, req.Email).First(&user).Error != nil

	// Always run bcrypt to prevent timing-based user enumeration
	passwordToCheck := user.Password
	if userNotFound {
		// Use a dummy hash so bcrypt still runs in constant time
		passwordToCheck = "$2a$10$dummyhashfortimingattttttttttttttttttttttttttttttttttt"
	}
	if userNotFound || !utils.CheckPassword(req.Password, passwordToCheck) {
		attempt.Success = false
		if err := db.Create(&attempt).Error; err != nil {
			utils.SysError("auth", fmt.Sprintf("记录登录失败尝试失败: %v", err))
		}
		utils.Unauthorized(c, "用户名或密码错误")
		return
	}

	if !user.IsActive {
		utils.Forbidden(c, "账户已被禁用")
		return
	}

	attempt.Success = true
	if err := db.Create(&attempt).Error; err != nil {
		utils.SysError("auth", fmt.Sprintf("记录登录成功尝试失败: %v", err))
	}

	// 更新最后登录时间
	if err := db.Model(&user).Update("last_login", time.Now()).Error; err != nil {
		utils.InternalError(c, "更新登录状态失败")
		return
	}

	// 记录登录历史
	loginIP := utils.GetRealClientIP(c)
	loginUA := c.GetHeader("User-Agent")
	loginLocation := utils.GetIPLocation(loginIP)
	if err := db.Create(&models.LoginHistory{
		UserID:      user.ID,
		IPAddress:   &loginIP,
		UserAgent:   &loginUA,
		Location:    &loginLocation,
		LoginStatus: "success",
	}).Error; err != nil {
		utils.InternalError(c, "记录登录历史失败")
		return
	}

	// 异常登录检测：比较上次登录IP与本次IP
	abnormalAlertGlobal := utils.IsBoolSettingDefault("abnormal_login_alert", true)
	if abnormalAlertGlobal && user.AbnormalLoginAlertEnabled {
		var lastLogin models.LoginHistory
		err := db.Where("user_id = ? AND login_status = 'success'", user.ID).
			Order("id DESC").Offset(1).First(&lastLogin).Error
		if err == nil && lastLogin.IPAddress != nil && *lastLogin.IPAddress != loginIP {
			go services.NotifyUser(user.ID, "abnormal_login", map[string]string{
				"ip": loginIP, "location": loginLocation,
				"last_ip": *lastLogin.IPAddress,
				"time":    time.Now().Format("2006-01-02 15:04:05"), "user_agent": loginUA,
			})
			go services.NotifyAdmin("abnormal_login", map[string]string{
				"username": user.Username,
				"ip":       loginIP,
				"location": loginLocation,
			})
		}
	}

	// Token 过期时间：优先使用管理员设置的 session_timeout_minutes
	accessExpireMinutes := config.AppConfig.AccessTokenExpireMinutes
	if v := utils.GetIntSetting("session_timeout_minutes", 0); v > 0 {
		accessExpireMinutes = v
	}
	accessToken, err := generateToken(user.ID, "access", time.Duration(accessExpireMinutes)*time.Minute)
	if err != nil {
		utils.InternalError(c, "生成访问令牌失败")
		return
	}
	refreshToken, err := generateToken(user.ID, "refresh", time.Duration(config.AppConfig.RefreshTokenExpireDays)*24*time.Hour)
	if err != nil {
		utils.InternalError(c, "生成刷新令牌失败")
		return
	}

	utils.Success(c, gin.H{
		"user": gin.H{
			"id": user.ID, "username": user.Username, "email": user.Email,
			"is_admin": user.IsAdmin, "nickname": user.Nickname, "avatar": user.Avatar,
			"balance": user.Balance, "theme": user.Theme,
		},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Logout 登出
func Logout(c *gin.Context) {
	// 黑名单 access token
	tokenString, _ := c.Get("token")
	if token, ok := tokenString.(string); ok {
		tokenHash := utils.SHA256Hash(token)
		if err := models.AddToBlacklist(database.GetDB(), tokenHash, c.GetUint("user_id"), time.Now().Add(24*time.Hour)); err != nil {
			utils.SysError("auth", fmt.Sprintf("登出加入黑名单失败: %v", err))
		}
	}

	// 同时黑名单 refresh token（如果客户端提供了）
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if c.ShouldBindJSON(&req) == nil && req.RefreshToken != "" {
		refreshHash := utils.SHA256Hash(req.RefreshToken)
		expiresAt := time.Now().Add(time.Duration(config.AppConfig.RefreshTokenExpireDays) * 24 * time.Hour)
		if err := models.AddToBlacklist(database.GetDB(), refreshHash, c.GetUint("user_id"), expiresAt); err != nil {
			utils.SysError("auth", fmt.Sprintf("登出 refresh token 加入黑名单失败: %v", err))
		}
	}

	utils.SuccessMessage(c, "已登出")
}

// RefreshToken 刷新 Token
func RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	token, err := jwt.ParseWithClaims(req.RefreshToken, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(config.GetSecretKey()), nil
	})
	if err != nil {
		utils.Unauthorized(c, "Refresh Token 无效")
		return
	}

	claims, ok := token.Claims.(*middleware.Claims)
	if !ok || !token.Valid || claims.Type != "refresh" {
		utils.Unauthorized(c, "Refresh Token 无效")
		return
	}

	// 检查 refresh token 是否在黑名单中
	tokenHash := utils.SHA256Hash(req.RefreshToken)
	if models.IsTokenBlacklisted(database.GetDB(), tokenHash) {
		utils.Unauthorized(c, "Refresh Token 已失效")
		return
	}

	var user models.User
	if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
		utils.Unauthorized(c, "用户不存在")
		return
	}

	if !user.IsActive {
		utils.Forbidden(c, "账户已被禁用")
		return
	}

	// 将旧的 refresh token 加入黑名单（防止重用）
	db := database.GetDB()
	expiresAt := time.Now().Add(time.Duration(config.AppConfig.RefreshTokenExpireDays) * 24 * time.Hour)
	if err := models.AddToBlacklist(db, tokenHash, user.ID, expiresAt); err != nil {
		utils.SysError("auth", fmt.Sprintf("refresh token 加入黑名单失败: %v", err))
	}

	accessToken, err := generateToken(user.ID, "access", time.Duration(config.AppConfig.AccessTokenExpireMinutes)*time.Minute)
	if err != nil {
		utils.InternalError(c, "生成访问令牌失败")
		return
	}
	newRefreshToken, err := generateToken(user.ID, "refresh", time.Duration(config.AppConfig.RefreshTokenExpireDays)*24*time.Hour)
	if err != nil {
		utils.InternalError(c, "生成刷新令牌失败")
		return
	}

	utils.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}

// SendVerificationCode 发送验证码
func SendVerificationCode(c *gin.Context) {
	var req struct {
		Email   string `json:"email" binding:"required,email"`
		Purpose string `json:"purpose"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if req.Purpose == "" {
		req.Purpose = "register"
	}

	db := database.GetDB()

	// 速率限制：同一邮箱 5 分钟内最多发送 3 次
	var recentCount int64
	since := time.Now().Add(-5 * time.Minute)
	db.Model(&models.VerificationCode{}).
		Where("email = ? AND purpose = ? AND created_at > ?", req.Email, req.Purpose, since).
		Count(&recentCount)
	if recentCount >= 3 {
		utils.TooManyRequests(c, "发送频率过高，请 5 分钟后再试")
		return
	}

	code := utils.GenerateVerificationCode()
	if err := db.Create(&models.VerificationCode{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Purpose:   req.Purpose,
	}).Error; err != nil {
		utils.InternalError(c, "验证码保存失败")
		return
	}

	// 发送验证码邮件
	subject, body := services.RenderEmail("verification", map[string]string{"code": code})
	go services.QueueEmail(req.Email, subject, body, "verification")
	utils.SuccessMessage(c, "验证码已发送")
}

// VerifyCode 验证验证码
func VerifyCode(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	var vc models.VerificationCode
	if err := database.GetDB().Where("email = ? AND code = ? AND used = 0 AND expires_at > ?",
		req.Email, req.Code, time.Now()).Order("created_at DESC").First(&vc).Error; err != nil {
		utils.BadRequest(c, "验证码无效或已过期")
		return
	}

	vc.MarkAsUsed()
	if err := database.GetDB().Save(&vc).Error; err != nil {
		utils.InternalError(c, "更新验证码状态失败")
		return
	}
	utils.SuccessMessage(c, "验证成功")
}

// ForgotPassword 忘记密码
func ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	// 速率限制：同一邮箱 15 分钟内最多发送 3 次重置请求
	var recentCount int64
	since := time.Now().Add(-15 * time.Minute)
	db.Model(&models.VerificationCode{}).
		Where("email = ? AND purpose = ? AND created_at > ?", req.Email, "reset_password", since).
		Count(&recentCount)
	if recentCount >= 3 {
		utils.TooManyRequests(c, "发送频率过高，请 15 分钟后再试")
		return
	}

	var user models.User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// 不暴露用户是否存在
		utils.SuccessMessage(c, "如果邮箱存在，重置链接已发送")
		return
	}

	code := utils.GenerateVerificationCode()
	if err := db.Create(&models.VerificationCode{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Purpose:   "reset_password",
	}).Error; err != nil {
		utils.InternalError(c, "验证码保存失败")
		return
	}

	// 发送重置密码邮件
	subject, body := services.RenderEmail("reset_password", map[string]string{"code": code})
	go services.QueueEmail(req.Email, subject, body, "reset_password")
	utils.SuccessMessage(c, "如果邮箱存在，重置链接已发送")
}

// ResetPassword 重置密码
func ResetPassword(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Code     string `json:"code" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if err := utils.ValidatePasswordStrength(req.Password); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	db := database.GetDB()

	var vc models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND purpose = ? AND used = 0 AND expires_at > ?",
		req.Email, req.Code, "reset_password", time.Now()).Order("created_at DESC").First(&vc).Error; err != nil {
		utils.BadRequest(c, "验证码无效或已过期")
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.InternalError(c, "密码加密失败")
		return
	}

	if err := db.Model(&models.User{}).Where("email = ?", req.Email).Update("password", hashedPassword).Error; err != nil {
		utils.InternalError(c, "更新密码失败")
		return
	}
	vc.MarkAsUsed()
	if err := db.Save(&vc).Error; err != nil {
		utils.InternalError(c, "更新验证码状态失败")
		return
	}

	utils.SuccessMessage(c, "密码重置成功")
}

// TelegramLogin Telegram 登录
func TelegramLogin(c *gin.Context) {
	var req services.TelegramLoginData
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	// Check if Telegram login is enabled
	if !utils.IsBoolSettingDefault("telegram_login_enabled", false) {
		utils.Forbidden(c, "Telegram 登录未启用")
		return
	}

	// Verify Telegram data
	if !services.VerifyTelegramLogin(&req) {
		utils.Unauthorized(c, "Telegram 验证失败")
		return
	}

	// Check auth_date is not too old (5 minutes)
	if time.Now().Unix()-req.AuthDate > 300 {
		utils.Unauthorized(c, "Telegram 授权已过期")
		return
	}

	db := database.GetDB()
	var user models.User

	// Find user by Telegram ID
	if err := db.Where("telegram_id = ?", req.ID).First(&user).Error; err != nil {
		utils.NotFound(c, "该 Telegram 账号未绑定任何用户，请先在设置中绑定")
		return
	}

	if !user.IsActive {
		utils.Forbidden(c, "账户已被禁用")
		return
	}

	// Update Telegram username if changed
	if req.Username != "" {
		if err := db.Model(&user).Update("telegram_username", req.Username).Error; err != nil {
			utils.InternalError(c, "更新 Telegram 用户名失败")
			return
		}
	}

	// Update last login
	if err := db.Model(&user).Update("last_login", time.Now()).Error; err != nil {
		utils.InternalError(c, "更新登录状态失败")
		return
	}

	// Generate tokens
	accessToken, err := generateToken(user.ID, "access", time.Duration(config.AppConfig.AccessTokenExpireMinutes)*time.Minute)
	if err != nil {
		utils.InternalError(c, "生成访问令牌失败")
		return
	}
	refreshToken, err := generateToken(user.ID, "refresh", time.Duration(config.AppConfig.RefreshTokenExpireDays)*24*time.Hour)
	if err != nil {
		utils.InternalError(c, "生成刷新令牌失败")
		return
	}

	utils.Success(c, gin.H{
		"user": gin.H{
			"id": user.ID, "username": user.Username, "email": user.Email,
			"is_admin": user.IsAdmin, "nickname": user.Nickname, "avatar": user.Avatar,
			"balance": user.Balance, "theme": user.Theme,
		},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// generateToken 生成 JWT Token
func generateToken(userID uint, tokenType string, expiry time.Duration) (string, error) {
	claims := middleware.Claims{
		UserID: userID,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.GetSecretKey()))
}
