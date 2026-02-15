package services

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
)

// userNotifySettingKey maps email template name to system-level setting key.
// Returns "" for templates that must always be sent (verification, reset_password).
func userNotifySettingKey(emailTemplate string) string {
	switch emailTemplate {
	case "welcome":
		return "user_notify_welcome"
	case "payment_success":
		return "user_notify_payment"
	case "expiry_reminder":
		return "user_notify_expiry"
	case "expiry_notice":
		return "user_notify_expired"
	case "subscription_reset":
		return "user_notify_reset"
	case "account_enabled", "account_disabled", "account_deleted":
		return "user_notify_account_status"
	case "unpaid_order":
		return "user_notify_unpaid_order"
	default:
		return ""
	}
}

// userPrefAllowed checks per-user notification preference for a given template.
func userPrefAllowed(user *models.User, emailTemplate string) bool {
	if !user.EmailNotifications {
		return false
	}
	switch emailTemplate {
	case "new_order", "payment_success":
		return user.NotifyOrder
	case "expiry_reminder", "expiry_notice":
		return user.NotifyExpiry
	case "subscription_reset", "account_enabled", "account_disabled", "account_deleted":
		return user.NotifySubscription
	case "abnormal_login":
		return user.AbnormalLoginAlertEnabled
	case "verification", "reset_password", "admin_create_user", "subscription":
		return true // always send
	default:
		return true
	}
}

// NotifyUser sends an email notification to a user, respecting system-level and user-level preferences.
func NotifyUser(userID uint, emailTemplate string, data map[string]string) {
	// Check system-level toggle
	sysKey := userNotifySettingKey(emailTemplate)
	if sysKey != "" {
		settings := utils.GetSettings(sysKey)
		if settings[sysKey] == "false" || settings[sysKey] == "0" {
			return
		}
	}
	db := database.GetDB()
	var user models.User
	if db.First(&user, userID).Error != nil {
		return
	}
	if !userPrefAllowed(&user, emailTemplate) {
		return
	}
	subject, body := RenderEmail(emailTemplate, data)
	go QueueEmail(user.Email, subject, body, emailTemplate)
}

// NotifyUserDirect sends an email to a specific address (for pre-registration or deleted users).
func NotifyUserDirect(email, emailTemplate string, data map[string]string) {
	sysKey := userNotifySettingKey(emailTemplate)
	if sysKey != "" {
		settings := utils.GetSettings(sysKey)
		if settings[sysKey] == "false" || settings[sysKey] == "0" {
			return
		}
	}
	subject, body := RenderEmail(emailTemplate, data)
	go QueueEmail(email, subject, body, emailTemplate)
}

// NotifyAdmin sends notifications to admin via all configured and enabled channels.
func NotifyAdmin(eventType string, data map[string]string) {
	settingKey := ""
	switch eventType {
	case "new_order":
		settingKey = "notify_new_order"
	case "payment_success":
		settingKey = "notify_payment_success"
	case "recharge_success":
		settingKey = "notify_recharge_success"
	case "new_ticket":
		settingKey = "notify_new_ticket"
	case "new_user", "admin_create_user":
		settingKey = "notify_new_user"
	case "subscription_reset":
		settingKey = "notify_subscription_reset"
	case "abnormal_login":
		settingKey = "notify_abnormal_login"
	case "unpaid_order":
		settingKey = "notify_unpaid_order"
	case "expiry_reminder":
		settingKey = "notify_expiry_reminder"
	default:
		return
	}

	settings := utils.GetSettings(
		settingKey,
		"notify_email_enabled", "notify_admin_email",
		"notify_telegram_enabled", "notify_telegram_bot_token", "notify_telegram_chat_id",
		"notify_bark_enabled", "notify_bark_server", "notify_bark_device_key",
		"site_name",
	)

	if settings[settingKey] != "true" && settings[settingKey] != "1" {
		return
	}

	siteName := settings["site_name"]
	if siteName == "" {
		siteName = "CBoard"
	}

	title, telegramMsg, barkMsg := buildNotifyMessage(siteName, eventType, data)

	// Email channel
	if email := settings["notify_admin_email"]; email != "" {
		enabled := settings["notify_email_enabled"]
		if enabled == "" || enabled == "true" || enabled == "1" {
			go QueueEmail(email, title, "<h3>"+title+"</h3><pre>"+barkMsg+"</pre>", "admin_notify")
		}
	}

	// Telegram channel
	botToken := settings["notify_telegram_bot_token"]
	chatID := settings["notify_telegram_chat_id"]
	if botToken != "" && chatID != "" {
		enabled := settings["notify_telegram_enabled"]
		if enabled == "true" || enabled == "1" {
			go sendTelegram(botToken, chatID, telegramMsg)
		}
	}

	// Bark channel
	barkServer := settings["notify_bark_server"]
	barkKey := settings["notify_bark_device_key"]
	if barkServer != "" && barkKey != "" {
		enabled := settings["notify_bark_enabled"]
		if enabled == "true" || enabled == "1" {
			go sendBark(barkServer, barkKey, title, barkMsg)
		}
	}
}

func buildNotifyMessage(siteName, eventType string, data map[string]string) (title, telegramBody, barkBody string) {
	now := time.Now().Format("2006-01-02 15:04:05")

	switch eventType {
	case "new_order":
		title = fmt.Sprintf("[%s] 📦 新订单", siteName)
		telegramBody = fmt.Sprintf(
			"📦 <b>新订单</b>\n\n"+
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
				"┃  📋 <b>订单详情</b>\n"+
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
				"🆔 <b>订单号</b>: <code>%s</code>\n"+
				"👤 <b>用户</b>: <code>%s</code>\n"+
				"📦 <b>套餐</b>: <b>%s</b>\n"+
				"💰 <b>金额</b>: <b>¥%s</b>\n"+
				"🕐 <b>时间</b>: %s",
			data["order_no"], data["username"], data["package_name"], data["amount"], now)
		barkBody = fmt.Sprintf(
			"📦 新订单\n\n"+
				"🆔 订单号: %s\n"+
				"👤 用户: %s\n"+
				"📦 套餐: %s\n"+
				"💰 金额: ¥%s\n"+
				"🕐 时间: %s",
			data["order_no"], data["username"], data["package_name"], data["amount"], now)

	case "payment_success":
		title = fmt.Sprintf("[%s] 🎉 支付成功", siteName)
		telegramBody = fmt.Sprintf(
			"🎉 <b>支付成功</b>\n\n"+
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
				"┃  📋 <b>订单详情</b>\n"+
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
				"🆔 <b>订单号</b>: <code>%s</code>\n"+
				"👤 <b>用户</b>: <code>%s</code>\n"+
				"📦 <b>套餐</b>: <b>%s</b>\n"+
				"💰 <b>金额</b>: <b>¥%s</b>\n"+
				"🕐 <b>时间</b>: %s\n\n"+
				"✅ 订单已自动处理\n"+
				"📦 订阅已激活",
			data["order_no"], data["username"], data["package_name"], data["amount"], now)
		barkBody = fmt.Sprintf(
			"🎉 支付成功\n\n"+
				"🆔 订单号: %s\n"+
				"👤 用户: %s\n"+
				"📦 套餐: %s\n"+
				"💰 金额: ¥%s\n"+
				"🕐 时间: %s\n\n"+
				"✅ 订单已自动处理\n"+
				"📦 订阅已激活",
			data["order_no"], data["username"], data["package_name"], data["amount"], now)

	case "recharge_success":
		title = fmt.Sprintf("[%s] 💰 充值成功", siteName)
		telegramBody = fmt.Sprintf(
			"💰 <b>充值成功</b>\n\n"+
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
				"┃  📋 <b>充值详情</b>\n"+
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
				"🆔 <b>充值单号</b>: <code>%s</code>\n"+
				"👤 <b>用户</b>: <code>%s</code>\n"+
				"💰 <b>金额</b>: <b>¥%s</b>\n"+
				"🕐 <b>时间</b>: %s",
			data["order_no"], data["username"], data["amount"], now)
		barkBody = fmt.Sprintf(
			"💰 充值成功\n\n"+
				"🆔 充值单号: %s\n"+
				"👤 用户: %s\n"+
				"💰 金额: ¥%s\n"+
				"🕐 时间: %s",
			data["order_no"], data["username"], data["amount"], now)

	case "new_ticket":
		title = fmt.Sprintf("[%s] 🎫 新工单", siteName)
		telegramBody = fmt.Sprintf(
			"🎫 <b>新工单</b>\n\n"+
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
				"┃  📋 <b>工单详情</b>\n"+
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
				"🆔 <b>工单号</b>: <code>%s</code>\n"+
				"👤 <b>用户</b>: <code>%s</code>\n"+
				"📝 <b>标题</b>: <b>%s</b>\n"+
				"🕐 <b>时间</b>: %s",
			data["ticket_no"], data["username"], data["title"], now)
		barkBody = fmt.Sprintf(
			"🎫 新工单\n\n"+
				"🆔 工单号: %s\n"+
				"👤 用户: %s\n"+
				"📝 标题: %s\n"+
				"🕐 时间: %s",
			data["ticket_no"], data["username"], data["title"], now)

	case "new_user":
		title = fmt.Sprintf("[%s] 👋 新用户注册", siteName)
		telegramBody = fmt.Sprintf(
			"👋 <b>新用户注册</b>\n\n"+
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
				"┃  📋 <b>用户详情</b>\n"+
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
				"👤 <b>用户名</b>: <code>%s</code>\n"+
				"📧 <b>邮箱</b>: <code>%s</code>\n"+
				"🕐 <b>时间</b>: %s\n\n"+
				"✅ 已自动创建默认订阅",
			data["username"], data["email"], now)
		barkBody = fmt.Sprintf(
			"👋 新用户注册\n\n"+
				"👤 用户名: %s\n"+
				"📧 邮箱: %s\n"+
				"🕐 时间: %s\n\n"+
				"✅ 已自动创建默认订阅",
			data["username"], data["email"], now)
	case "admin_create_user":
		title = fmt.Sprintf("[%s] 📋 管理员创建用户", siteName)
		telegramBody = fmt.Sprintf(
			"📋 <b>管理员创建用户</b>\n\n"+
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
				"┃  📋 <b>用户详情</b>\n"+
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
				"👤 <b>用户名</b>: <code>%s</code>\n"+
				"📧 <b>邮箱</b>: <code>%s</code>\n"+
				"🕐 <b>时间</b>: %s",
			data["username"], data["email"], now)
		barkBody = fmt.Sprintf(
			"📋 管理员创建用户\n\n"+
				"👤 用户名: %s\n"+
				"📧 邮箱: %s\n"+
				"🕐 时间: %s",
			data["username"], data["email"], now)

	case "subscription_reset":
		title = fmt.Sprintf("[%s] 🔄 订阅重置", siteName)
		telegramBody = fmt.Sprintf(
			"🔄 <b>订阅重置</b>\n\n"+
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
				"┃  📋 <b>重置详情</b>\n"+
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
				"👤 <b>用户</b>: <code>%s</code>\n"+
				"🔧 <b>操作者</b>: <code>%s</code>\n"+
				"🕐 <b>时间</b>: %s\n\n"+
				"⚠️ 旧地址已失效",
			data["username"], data["reset_by"], now)
		barkBody = fmt.Sprintf(
			"🔄 订阅重置\n\n"+
				"👤 用户: %s\n"+
				"🔧 操作者: %s\n"+
				"🕐 时间: %s\n\n"+
				"⚠️ 旧地址已失效",
			data["username"], data["reset_by"], now)

	case "abnormal_login":
		title = fmt.Sprintf("[%s] ⚠️ 异常登录", siteName)
		telegramBody = fmt.Sprintf(
			"⚠️ <b>异常登录</b>\n\n"+
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
				"┃  📋 <b>登录详情</b>\n"+
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
				"👤 <b>用户</b>: <code>%s</code>\n"+
				"🌐 <b>IP</b>: <code>%s</code>\n"+
				"📍 <b>位置</b>: <b>%s</b>\n"+
				"🕐 <b>时间</b>: %s",
			data["username"], data["ip"], data["location"], now)
		barkBody = fmt.Sprintf(
			"⚠️ 异常登录\n\n"+
				"👤 用户: %s\n"+
				"🌐 IP: %s\n"+
				"📍 位置: %s\n"+
				"🕐 时间: %s",
			data["username"], data["ip"], data["location"], now)
	case "unpaid_order":
		title = fmt.Sprintf("[%s] ⏳ 未支付订单", siteName)
		telegramBody = fmt.Sprintf(
			"⏳ <b>未支付订单</b>\n\n"+
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
				"┃  📋 <b>订单详情</b>\n"+
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
				"🆔 <b>订单号</b>: <code>%s</code>\n"+
				"👤 <b>用户</b>: <code>%s</code>\n"+
				"💰 <b>金额</b>: <b>¥%s</b>\n"+
				"🕐 <b>时间</b>: %s",
			data["order_no"], data["username"], data["amount"], now)
		barkBody = fmt.Sprintf(
			"⏳ 未支付订单\n\n"+
				"🆔 订单号: %s\n"+
				"👤 用户: %s\n"+
				"💰 金额: ¥%s\n"+
				"🕐 时间: %s",
			data["order_no"], data["username"], data["amount"], now)

	case "expiry_reminder":
		title = fmt.Sprintf("[%s] ⏰ 订阅到期提醒", siteName)
		telegramBody = fmt.Sprintf(
			"⏰ <b>订阅到期提醒</b>\n\n"+
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
				"┃  📋 <b>到期详情</b>\n"+
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
				"👤 <b>用户</b>: <code>%s</code>\n"+
				"⏰ <b>到期时间</b>: <b>%s</b>",
			data["username"], data["expire_time"])
		barkBody = fmt.Sprintf(
			"⏰ 订阅到期提醒\n\n"+
				"👤 用户: %s\n"+
				"⏰ 到期时间: %s",
			data["username"], data["expire_time"])

	default:
		title = fmt.Sprintf("[%s] 通知", siteName)
		telegramBody = data["message"]
		barkBody = data["message"]
	}

	return title, telegramBody, barkBody
}

// SendTestTelegram sends a test message via Telegram using saved settings.
func SendTestTelegram() error {
	settings := utils.GetSettings("notify_telegram_bot_token", "notify_telegram_chat_id", "site_name")
	botToken := settings["notify_telegram_bot_token"]
	chatID := settings["notify_telegram_chat_id"]
	if botToken == "" || chatID == "" {
		return fmt.Errorf("请先配置 Telegram Bot Token 和 Chat ID")
	}
	siteName := settings["site_name"]
	if siteName == "" {
		siteName = "CBoard"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(
		"✅ <b>Telegram 通知测试成功</b>\n\n"+
			"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n"+
			"┃  📋 <b>测试信息</b>\n"+
			"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n"+
			"🏷️ <b>站点</b>: <b>%s</b>\n"+
			"🕐 <b>时间</b>: %s\n\n"+
			"📡 通知服务运行正常",
		siteName, now)
	return sendTelegramSync(botToken, chatID, msg)
}

func sendTelegram(botToken, chatID, message string) {
	_ = sendTelegramSync(botToken, chatID, message)
}

func sendTelegramSync(botToken, chatID, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", message)
	form.Set("parse_mode", "HTML")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(apiURL, form)
	if err != nil {
		log.Printf("[Notify] Telegram 发送失败: %v", err)
		utils.SysError("notify", "Telegram 发送失败", err.Error())
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[Notify] Telegram 返回状态码: %d", resp.StatusCode)
		utils.SysWarn("notify", fmt.Sprintf("Telegram 返回状态码: %d", resp.StatusCode))
		return fmt.Errorf("Telegram 返回状态码: %d", resp.StatusCode)
	}
	return nil
}

func sendBark(serverURL, deviceKey, title, body string) {
	serverURL = strings.TrimRight(serverURL, "/")
	reqURL := fmt.Sprintf("%s/%s/%s/%s",
		serverURL, url.PathEscape(deviceKey),
		url.PathEscape(title), url.PathEscape(body))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		log.Printf("[Notify] Bark 发送失败: %v", err)
		utils.SysError("notify", "Bark 发送失败", err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[Notify] Bark 返回状态码: %d", resp.StatusCode)
		utils.SysWarn("notify", fmt.Sprintf("Bark 返回状态码: %d", resp.StatusCode))
	}
}
