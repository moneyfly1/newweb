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

// NotifyUser sends an email notification to a user, respecting their notification preferences.
// emailTemplate is the RenderEmail template name, data is passed to RenderEmail.
func NotifyUser(userID uint, emailTemplate string, data map[string]string) {
	db := database.GetDB()
	var user models.User
	if db.First(&user, userID).Error != nil {
		return
	}
	if !user.EmailNotifications {
		return
	}
	subject, body := RenderEmail(emailTemplate, data)
	go QueueEmail(user.Email, subject, body, emailTemplate)
}

// NotifyUserDirect sends an email to a specific address (for pre-registration or deleted users).
func NotifyUserDirect(email, emailTemplate string, data map[string]string) {
	subject, body := RenderEmail(emailTemplate, data)
	go QueueEmail(email, subject, body, emailTemplate)
}

// NotifyAdmin sends notifications to admin via all configured channels.
func NotifyAdmin(eventType string, data map[string]string) {
	settingKey := ""
	switch eventType {
	case "new_order", "payment_success", "recharge_success":
		settingKey = "notify_new_order"
	case "new_ticket":
		settingKey = "notify_new_ticket"
	case "new_user", "admin_create_user":
		settingKey = "notify_new_user"
	case "subscription_reset":
		settingKey = "notify_new_order"
	case "abnormal_login":
		settingKey = "notify_new_user"
	default:
		return
	}

	settings := utils.GetSettings(
		settingKey, "notify_admin_email",
		"notify_telegram_bot_token", "notify_telegram_chat_id",
		"notify_bark_server", "notify_bark_device_key",
		"site_name",
	)

	if settings[settingKey] != "true" && settings[settingKey] != "1" {
		return
	}

	siteName := settings["site_name"]
	if siteName == "" {
		siteName = "CBoard"
	}

	title, body := buildNotifyMessage(siteName, eventType, data)

	if email := settings["notify_admin_email"]; email != "" {
		go QueueEmail(email, title, "<h3>"+title+"</h3><pre>"+body+"</pre>", "admin_notify")
	}

	botToken := settings["notify_telegram_bot_token"]
	chatID := settings["notify_telegram_chat_id"]
	if botToken != "" && chatID != "" {
		go sendTelegram(botToken, chatID, fmt.Sprintf("*%s*\n%s", title, body))
	}

	barkServer := settings["notify_bark_server"]
	barkKey := settings["notify_bark_device_key"]
	if barkServer != "" && barkKey != "" {
		go sendBark(barkServer, barkKey, title, body)
	}
}

func buildNotifyMessage(siteName, eventType string, data map[string]string) (string, string) {
	switch eventType {
	case "new_order":
		return fmt.Sprintf("[%s] 新订单", siteName),
			fmt.Sprintf("用户: %s\n订单号: %s\n套餐: %s\n金额: ¥%s", data["username"], data["order_no"], data["package_name"], data["amount"])
	case "payment_success":
		return fmt.Sprintf("[%s] 支付成功", siteName),
			fmt.Sprintf("用户: %s\n订单号: %s\n套餐: %s\n金额: ¥%s", data["username"], data["order_no"], data["package_name"], data["amount"])
	case "recharge_success":
		return fmt.Sprintf("[%s] 充值成功", siteName),
			fmt.Sprintf("用户: %s\n充值单号: %s\n金额: ¥%s", data["username"], data["order_no"], data["amount"])
	case "new_ticket":
		return fmt.Sprintf("[%s] 新工单", siteName),
			fmt.Sprintf("用户: %s\n工单号: %s\n标题: %s", data["username"], data["ticket_no"], data["title"])
	case "new_user":
		return fmt.Sprintf("[%s] 新用户注册", siteName),
			fmt.Sprintf("用户名: %s\n邮箱: %s", data["username"], data["email"])
	case "admin_create_user":
		return fmt.Sprintf("[%s] 管理员创建用户", siteName),
			fmt.Sprintf("用户名: %s\n邮箱: %s", data["username"], data["email"])
	case "subscription_reset":
		return fmt.Sprintf("[%s] 订阅重置", siteName),
			fmt.Sprintf("用户: %s\n操作: %s", data["username"], data["reset_by"])
	case "abnormal_login":
		return fmt.Sprintf("[%s] 异常登录", siteName),
			fmt.Sprintf("用户: %s\nIP: %s\n位置: %s", data["username"], data["ip"], data["location"])
	default:
		return fmt.Sprintf("[%s] 通知", siteName), data["message"]
	}
}

func sendTelegram(botToken, chatID, message string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", message)
	form.Set("parse_mode", "Markdown")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(apiURL, form)
	if err != nil {
		log.Printf("[Notify] Telegram 发送失败: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[Notify] Telegram 返回状态码: %d", resp.StatusCode)
	}
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
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[Notify] Bark 返回状态码: %d", resp.StatusCode)
	}
}
