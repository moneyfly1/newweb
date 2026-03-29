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
			// 使用模板系统构建管理员通知邮件
			builder := NewEmailTemplateBuilder()
			emailBody := builder.GetAdminNotificationTemplate(eventType, title, barkMsg, convertToInterfaceMap(data))
			go QueueEmail(email, title, emailBody, "admin_notify")
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

// buildNotifyMessage 使用模板系统构建通知消息
func buildNotifyMessage(siteName, eventType string, data map[string]string) (title, telegramBody, barkBody string) {
	// 获取模板
	template := GetNotifyTemplate(eventType)
	if template == nil {
		// 回退到默认消息
		title = fmt.Sprintf("[%s] 通知", siteName)
		msg := data["message"]
		if msg == "" {
			msg = "系统通知"
		}
		return title, msg, msg
	}

	// 渲染标题
	title = RenderNotifyTitle(siteName, template)

	// 渲染 Telegram 消息
	telegramBody = RenderTelegramMessage(template, data)

	// 渲染 Bark 消息
	barkBody = RenderBarkMessage(template, data)

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

	// 使用模板系统构建测试消息
	var sb strings.Builder
	sb.WriteString("✅ <b>Telegram 通知测试成功</b>\n\n")
	sb.WriteString("┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n")
	sb.WriteString("┃  📋 <b>测试信息</b>\n")
	sb.WriteString("┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n")
	sb.WriteString(fmt.Sprintf("🏷️ <b>站点</b>: <b>%s</b>\n", siteName))
	sb.WriteString(fmt.Sprintf("🕐 <b>时间</b>: %s\n\n", now))
	sb.WriteString("📡 通知服务运行正常")

	return sendTelegramSync(botToken, chatID, sb.String())
}

// SendTestBark sends a test message via Bark using saved settings.
func SendTestBark() error {
	settings := utils.GetSettings("notify_bark_server", "notify_bark_device_key", "site_name")
	barkServer := settings["notify_bark_server"]
	barkKey := settings["notify_bark_device_key"]
	if barkServer == "" || barkKey == "" {
		return fmt.Errorf("请先配置 Bark 服务器地址和 Device Key")
	}
	siteName := settings["site_name"]
	if siteName == "" {
		siteName = "CBoard"
	}
	now := time.Now().Format("2006-01-02 15:04:05")

	title := fmt.Sprintf("[%s] ✅ Bark 测试", siteName)
	body := fmt.Sprintf("🏷️ 站点: %s\n🕐 时间: %s\n\n📡 通知服务运行正常", siteName, now)

	barkServer = strings.TrimRight(barkServer, "/")
	reqURL := fmt.Sprintf("%s/%s/%s/%s",
		barkServer, url.PathEscape(barkKey),
		url.PathEscape(title), url.PathEscape(body))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		return fmt.Errorf("Bark 发送失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bark 返回状态码: %d", resp.StatusCode)
	}
	return nil
}

func sendTelegram(botToken, chatID, message string) {
	if err := sendTelegramSync(botToken, chatID, message); err != nil {
		log.Printf("[Notify] Telegram 异步发送失败: %v", err)
	}
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

// convertToInterfaceMap 将 map[string]string 转换为 map[string]interface{}
func convertToInterfaceMap(data map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		result[k] = v
	}
	return result
}
