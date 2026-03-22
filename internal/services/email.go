package services

import (
	"crypto/tls"
	"fmt"
	"html"
	"log"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
)

// SMTPConfig holds SMTP connection parameters
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// GetSMTPConfig reads SMTP settings from system_configs table,
// falling back to app config env vars.
func GetSMTPConfig() (*SMTPConfig, error) {
	m := utils.GetSettings("smtp_host", "smtp_port", "smtp_username", "smtp_password", "smtp_from", "smtp_from_email", "smtp_from_name")

	host := m["smtp_host"]
	if host == "" {
		return nil, fmt.Errorf("SMTP 未配置: smtp_host 为空")
	}

	port := 587
	if p, err := strconv.Atoi(m["smtp_port"]); err == nil && p > 0 {
		port = p
	}

	from := m["smtp_from_email"]
	if from == "" {
		from = m["smtp_from"]
	}
	if from == "" {
		from = m["smtp_username"]
	}
	if name := m["smtp_from_name"]; name != "" && from != "" && !strings.Contains(from, "<") {
		from = fmt.Sprintf("%s <%s>", name, from)
	}

	return &SMTPConfig{
		Host:     host,
		Port:     port,
		Username: m["smtp_username"],
		Password: m["smtp_password"],
		From:     from,
	}, nil
}

// SendEmail sends an email via SMTP. It tries TLS first (port 465),
// then STARTTLS for other ports.
func SendEmail(to, subject, body string) error {
	cfg, err := GetSMTPConfig()
	if err != nil {
		return err
	}
	return SendEmailWithConfig(cfg, to, subject, body)
}

// SendEmailWithConfig sends an email using the provided SMTP config.
func SendEmailWithConfig(cfg *SMTPConfig, to, subject, body string) error {
	headerFrom := cfg.From
	if headerFrom == "" {
		headerFrom = cfg.Username
	}

	// Extract bare email for SMTP envelope (MAIL FROM)
	envelopeFrom := headerFrom
	if idx := strings.Index(envelopeFrom, "<"); idx != -1 {
		envelopeFrom = strings.TrimRight(envelopeFrom[idx+1:], ">")
	}

	msg := buildMIME(headerFrom, to, subject, body)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	if cfg.Port == 465 {
		// Implicit TLS
		tlsCfg := &tls.Config{ServerName: cfg.Host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("TLS 连接失败: %w", err)
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return fmt.Errorf("SMTP 客户端创建失败: %w", err)
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
		if err = client.Mail(envelopeFrom); err != nil {
			return err
		}
		if err = client.Rcpt(to); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(msg))
		if err != nil {
			return err
		}
		return w.Close()
	}

	// STARTTLS (port 587 / 25)
	return smtp.SendMail(addr, auth, envelopeFrom, []string{to}, []byte(msg))
}

// sanitizeHeader 清除 MIME header 中的换行符，防止邮件头注入
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

func buildMIME(from, to, subject, body string) string {
	var sb strings.Builder
	sb.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	sb.WriteString("To: " + sanitizeHeader(to) + "\r\n")
	sb.WriteString("Subject: " + sanitizeHeader(subject) + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return sb.String()
}

// QueueEmail inserts an email into the email_queue table.
// A background worker or the caller can process it later.
func QueueEmail(toEmail, subject, content, emailType string) {
	db := database.GetDB()
	if err := db.Create(&models.EmailQueue{
		ToEmail:     toEmail,
		Subject:     subject,
		Content:     content,
		ContentType: "html",
		EmailType:   emailType,
		Status:      "pending",
		MaxRetries:  3,
	}).Error; err != nil {
		utils.SysError("email", fmt.Sprintf("写入邮件队列失败: to=%s type=%s err=%v", toEmail, emailType, err))
	}
}

// ProcessEmailQueue tries to send all pending emails in the queue.
func ProcessEmailQueue() {
	db := database.GetDB()
	var emails []models.EmailQueue
	db.Where("status = ? AND retry_count < max_retries", "pending").
		Order("created_at ASC").Limit(50).Find(&emails)

	if len(emails) == 0 {
		return
	}
	log.Printf("[EmailQueue] 发现 %d 封待发送邮件", len(emails))

	for i := range emails {
		eq := &emails[i]
		err := SendEmail(eq.ToEmail, eq.Subject, eq.Content)
		now := time.Now()
		if err != nil {
			errMsg := err.Error()
			eq.ErrorMessage = &errMsg
			eq.RetryCount++
			if eq.RetryCount >= eq.MaxRetries {
				eq.Status = "failed"
			}
			if saveErr := db.Save(eq).Error; saveErr != nil {
				utils.SysError("email", fmt.Sprintf("更新邮件队列失败: id=%d err=%v", eq.ID, saveErr))
			}
			log.Printf("[EmailQueue] 发送失败 #%d -> %s: %s", eq.ID, eq.ToEmail, errMsg)
			utils.SysError("email", fmt.Sprintf("发送失败 -> %s", eq.ToEmail), errMsg)
		} else {
			eq.Status = "sent"
			eq.SentAt = &now
			if saveErr := db.Save(eq).Error; saveErr != nil {
				utils.SysError("email", fmt.Sprintf("更新邮件队列失败: id=%d err=%v", eq.ID, saveErr))
			}
			log.Printf("[EmailQueue] 发送成功 #%d -> %s", eq.ID, eq.ToEmail)
			utils.SysInfo("email", fmt.Sprintf("发送成功 -> %s (%s)", eq.ToEmail, eq.EmailType))
		}
	}
}

// ── Email templates ──

// RenderEmail renders a named email template with the given data.
func RenderEmail(templateName string, data map[string]string) (subject, htmlBody string) {
	builder := NewEmailTemplateBuilder()
	siteName := utils.GetSetting("site_name")
	if siteName == "" {
		siteName = "CBoard"
	}

	switch templateName {
	case "verification":
		subject = fmt.Sprintf("邮箱验证码 - %s", siteName)
		htmlBody = builder.GetVerificationCodeTemplate(data["username"], data["code"])
	case "reset_password":
		subject = fmt.Sprintf("密码重置 - %s", siteName)
		// Use verification code template for password reset
		htmlBody = builder.GetPasswordResetVerificationCodeTemplate(data["username"], data["code"])
	case "welcome":
		subject = fmt.Sprintf("欢迎加入 %s", siteName)
		loginURL := builder.GetBaseURL() + "/login"
		htmlBody = builder.GetWelcomeTemplate(data["username"], data["email"], loginURL, true, "")
	case "subscription":
		subject = fmt.Sprintf("您的订阅信息 - %s", siteName)
		// Parse remaining days, device limit, current devices from data or use defaults
		remainingDays := 30
		deviceLimit := 5
		currentDevices := 0
		htmlBody = builder.GetSubscriptionTemplate(data["username"], data["universal_url"], data["clash_url"], data["expire_time"], remainingDays, deviceLimit, currentDevices)
	case "payment_success":
		subject = fmt.Sprintf("支付成功 - %s", siteName)
		// Parse amount from string to float64
		amount := 0.0
		if amountStr := data["amount"]; amountStr != "" {
			if parsed, err := strconv.ParseFloat(amountStr, 64); err == nil {
				amount = parsed
			}
		}
		paymentTime := time.Now().Format("2006-01-02 15:04:05")
		htmlBody = builder.GetPaymentSuccessTemplate(data["username"], data["order_no"], data["package_name"], amount, "支付宝", paymentTime)
	case "recharge_success":
		subject = fmt.Sprintf("充值成功 - %s", siteName)
		// Use payment success template for recharge
		amount := 0.0
		if amountStr := data["amount"]; amountStr != "" {
			if parsed, err := strconv.ParseFloat(amountStr, 64); err == nil {
				amount = parsed
			}
		}
		paymentTime := time.Now().Format("2006-01-02 15:04:05")
		htmlBody = builder.GetPaymentSuccessTemplate(data["username"], data["order_no"], "余额充值", amount, "支付宝", paymentTime)
	case "expiry_reminder":
		subject = fmt.Sprintf("%s - 订阅即将到期提醒", siteName)
		// Parse days from string to int
		remainingDays := 3
		if daysStr := data["days"]; daysStr != "" {
			if parsed, err := strconv.Atoi(daysStr); err == nil {
				remainingDays = parsed
			}
		}
		htmlBody = builder.GetExpirationReminderTemplate(data["username"], "订阅套餐", data["expire_time"], remainingDays, 5, 0, false)
	case "expiry_notice":
		subject = fmt.Sprintf("%s - 订阅已过期", siteName)
		htmlBody = builder.GetExpirationReminderTemplate(data["username"], "订阅套餐", data["expire_time"], 0, 5, 0, true)
	case "test":
		subject = fmt.Sprintf("%s - 测试邮件", siteName)
		htmlBody = builder.GetBroadcastNotificationTemplate("测试邮件", "<p>如果您收到此邮件，说明 SMTP 配置正确。这是一封测试邮件，无需任何操作。</p>")
	case "admin_create_user":
		subject = fmt.Sprintf("账户已创建 - %s", siteName)
		// Use welcome template with password reset instruction
		loginURL := builder.GetBaseURL() + "/login"
		htmlBody = builder.GetWelcomeTemplate(data["username"], data["email"], loginURL, false, "")
	case "account_disabled":
		subject = fmt.Sprintf("账户已被禁用 - %s", siteName)
		htmlBody = builder.GetBroadcastNotificationTemplate("账户已禁用", fmt.Sprintf("<p>您好，您的 %s 账户已被管理员禁用。</p><p>您的账户已无法登录和使用服务。如有疑问，请联系客服。</p>", siteName))
	case "account_enabled":
		subject = fmt.Sprintf("账户已恢复 - %s", siteName)
		htmlBody = builder.GetBroadcastNotificationTemplate("账户已恢复", fmt.Sprintf("<p>您好，您的 %s 账户已被管理员恢复启用。</p><p>✅ 您现在可以正常登录和使用服务了。</p>", siteName))
	case "account_deleted":
		subject = fmt.Sprintf("账户已删除 - %s", siteName)
		deletionDate := time.Now().Format("2006-01-02")
		htmlBody = builder.GetAccountDeletionTemplate(data["username"], deletionDate, "管理员删除", "30天")
	case "subscription_reset":
		subject = fmt.Sprintf("订阅地址已重置 - %s", siteName)
		resetTime := time.Now().Format("2006-01-02 15:04:05")
		htmlBody = builder.GetSubscriptionResetTemplate(data["username"], data["universal_url"], data["clash_url"], data["expire_time"], resetTime, data["reset_by"])
	case "abnormal_login":
		subject = fmt.Sprintf("异常登录提醒 - %s", siteName)
		htmlBody = builder.GetAbnormalLoginAlertTemplate(data["username"], data["time"], data["ip"], data["location"], true, true)
	case "unpaid_order":
		subject = fmt.Sprintf("您有未支付的订单 - %s", siteName)
		// Use order confirmation template
		amount := 0.0
		if amountStr := data["amount"]; amountStr != "" {
			if parsed, err := strconv.ParseFloat(amountStr, 64); err == nil {
				amount = parsed
			}
		}
		orderTime := time.Now().Format("2006-01-02 15:04:05")
		htmlBody = builder.GetOrderConfirmationTemplate(data["username"], data["order_no"], data["package_name"], amount, "待支付", orderTime)
	case "new_order":
		subject = fmt.Sprintf("新订单通知 - %s", siteName)
		// Use order confirmation template
		amount := 0.0
		if amountStr := data["amount"]; amountStr != "" {
			if parsed, err := strconv.ParseFloat(amountStr, 64); err == nil {
				amount = parsed
			}
		}
		orderTime := time.Now().Format("2006-01-02 15:04:05")
		htmlBody = builder.GetOrderConfirmationTemplate(data["username"], data["order_no"], data["package_name"], amount, "待支付", orderTime)
	default:
		subject = fmt.Sprintf("通知 - %s", siteName)
		message := data["message"]
		if message == "" {
			message = "您有一条新通知"
		}
		htmlBody = builder.GetBroadcastNotificationTemplate("系统通知", "<p>"+html.EscapeString(message)+"</p>")
	}
	return
}
