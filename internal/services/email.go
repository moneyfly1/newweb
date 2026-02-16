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

func buildMIME(from, to, subject, body string) string {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
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
	db.Create(&models.EmailQueue{
		ToEmail:     toEmail,
		Subject:     subject,
		Content:     content,
		ContentType: "html",
		EmailType:   emailType,
		Status:      "pending",
		MaxRetries:  3,
	})
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
			db.Save(eq)
			log.Printf("[EmailQueue] 发送失败 #%d -> %s: %s", eq.ID, eq.ToEmail, errMsg)
			utils.SysError("email", fmt.Sprintf("发送失败 -> %s", eq.ToEmail), errMsg)
		} else {
			eq.Status = "sent"
			eq.SentAt = &now
			db.Save(eq)
			log.Printf("[EmailQueue] 发送成功 #%d -> %s", eq.ID, eq.ToEmail)
			utils.SysInfo("email", fmt.Sprintf("发送成功 -> %s (%s)", eq.ToEmail, eq.EmailType))
		}
	}
}

// ── Email templates ──

// RenderEmail renders a named email template with the given data.
func RenderEmail(templateName string, data map[string]string) (subject, htmlBody string) {
	siteName := utils.GetSetting("site_name")
	if siteName == "" {
		siteName = "CBoard"
	}
	domain := GetSiteURL()

	var title, content, btnText, btnLink string

	switch templateName {
	case "verification":
		subject = fmt.Sprintf("邮箱验证码 - %s", siteName)
		title = "邮箱验证码"
		code := html.EscapeString(data["code"])
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您正在进行邮箱验证操作。</p>
<div style="text-align:center;margin:24px 0">
  <span style="display:inline-block;font-size:32px;font-weight:bold;letter-spacing:8px;color:#4F46E5;background:#F0EFFF;padding:12px 24px;border-radius:8px">%s</span>
</div>
<p style="font-size:14px;color:#666;margin:0">验证码有效期 5 分钟，请勿泄露给他人。</p>`, code)
	case "reset_password":
		subject = fmt.Sprintf("密码重置 - %s", siteName)
		title = "密码重置"
		code := html.EscapeString(data["code"])
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您正在进行密码重置操作。</p>
<div style="text-align:center;margin:24px 0">
  <span style="display:inline-block;font-size:32px;font-weight:bold;letter-spacing:8px;color:#4F46E5;background:#F0EFFF;padding:12px 24px;border-radius:8px">%s</span>
</div>
<p style="font-size:14px;color:#666;margin:0">验证码有效期 15 分钟。如果这不是您的操作，请忽略此邮件。</p>`, code)
	// TEMPLATE_PLACEHOLDER_1
	case "welcome":
		subject = fmt.Sprintf("欢迎加入 %s", siteName)
		title = "欢迎加入"
		username := html.EscapeString(data["username"])
		email := html.EscapeString(data["email"])
		password := html.EscapeString(data["password"])
		loginURL := domain + "/login"
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">Hi %s，欢迎加入 %s！您的账户已创建成功。</p>
<div style="background:#F9FAFB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">邮箱：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0 0 8px">密码：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0">登录地址：<a href="%s" style="color:#4F46E5">%s</a></p>
</div>
<p style="font-size:13px;color:#E11D48;margin:0">⚠️ 请妥善保管您的账户信息，建议登录后修改密码。</p>`, username, siteName, email, password, loginURL, loginURL)
		btnText = "立即登录"
		btnLink = loginURL
	case "subscription":
		subject = fmt.Sprintf("您的订阅信息 - %s", siteName)
		title = "订阅信息"
		clashURL := html.EscapeString(data["clash_url"])
		universalURL := html.EscapeString(data["universal_url"])
		expireTime := html.EscapeString(data["expire_time"])
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，以下是您的订阅信息：</p>
<div style="background:#F9FAFB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:13px;color:#888;margin:0 0 4px">Clash 订阅链接</p>
  <p style="font-size:13px;color:#333;word-break:break-all;margin:0 0 12px;background:#fff;padding:8px;border-radius:4px;border:1px solid #E5E7EB"><code>%s</code></p>
  <p style="font-size:13px;color:#888;margin:0 0 4px">通用订阅链接</p>
  <p style="font-size:13px;color:#333;word-break:break-all;margin:0;background:#fff;padding:8px;border-radius:4px;border:1px solid #E5E7EB"><code>%s</code></p>
</div>
<p style="font-size:14px;color:#666;margin:0 0 8px">到期时间：<strong>%s</strong></p>
<p style="font-size:13px;color:#999;margin:0">请妥善保管，不要泄露给他人。</p>`, clashURL, universalURL, expireTime)
		btnText = "查看订阅"
		btnLink = domain + "/subscription"
	case "payment_success":
		subject = fmt.Sprintf("支付成功 - %s", siteName)
		title = "支付成功"
		orderNo := html.EscapeString(data["order_no"])
		amount := html.EscapeString(data["amount"])
		packageName := html.EscapeString(data["package_name"])
		subURL := data["subscription_url"]
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的订单已支付成功！</p>
<div style="background:#F0FDF4;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">订单号：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0 0 8px">套餐：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0">金额：<strong>¥%s</strong></p>
</div>`, orderNo, packageName, amount)
		if subURL != "" {
			escapedURL := html.EscapeString(subURL)
			content += fmt.Sprintf(`<div style="background:#F9FAFB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:13px;color:#888;margin:0 0 4px">您的订阅地址</p>
  <p style="font-size:13px;color:#333;word-break:break-all;margin:0;background:#fff;padding:8px;border-radius:4px;border:1px solid #E5E7EB"><code>%s</code></p>
</div>`, escapedURL)
		}
		content += `<p style="font-size:14px;color:#666;margin:0">订阅已自动激活，感谢您的支持！</p>`
		btnText = "查看订阅"
		btnLink = domain + "/subscription"
	// TEMPLATE_PLACEHOLDER_2
	case "recharge_success":
		subject = fmt.Sprintf("充值成功 - %s", siteName)
		title = "充值成功"
		orderNo := html.EscapeString(data["order_no"])
		amount := html.EscapeString(data["amount"])
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的充值已成功到账！</p>
<div style="background:#F0FDF4;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">充值单号：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0">充值金额：<strong>¥%s</strong></p>
</div>
<p style="font-size:14px;color:#666;margin:0">余额已自动到账，感谢您的支持！</p>`, orderNo, amount)
		btnText = "查看余额"
		btnLink = domain + "/"
	case "expiry_reminder":
		days := html.EscapeString(data["days"])
		expireTime := html.EscapeString(data["expire_time"])
		subject = fmt.Sprintf("%s - 订阅即将到期提醒", siteName)
		title = "订阅到期提醒"
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的 %s 订阅将在 <strong>%s 天</strong>后到期。</p>
<div style="background:#FFFBEB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#92400E;margin:0">⏰ 到期时间：<strong>%s</strong></p>
</div>
<p style="font-size:14px;color:#666;margin:0">请及时续费以免服务中断。</p>`, siteName, days, expireTime)
		btnText = "立即续费"
		btnLink = domain + "/shop"
	case "expiry_notice":
		expireTime := html.EscapeString(data["expire_time"])
		subject = fmt.Sprintf("%s - 订阅已过期", siteName)
		title = "订阅已过期"
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的 %s 订阅已于 <strong>%s</strong> 过期。</p>
<div style="background:#FEF2F2;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#991B1B;margin:0">❌ 您的服务已暂停，续费后将自动恢复。</p>
</div>
<p style="font-size:14px;color:#666;margin:0">请续费以恢复服务。</p>`, siteName, expireTime)
		btnText = "立即续费"
		btnLink = domain + "/shop"
	case "test":
		subject = fmt.Sprintf("%s - 测试邮件", siteName)
		title = "测试邮件"
		content = `<p style="font-size:15px;color:#333;margin:0 0 16px">如果您收到此邮件，说明 SMTP 配置正确。</p>
<p style="font-size:14px;color:#666;margin:0">这是一封测试邮件，无需任何操作。</p>`
		btnText = "访问面板"
		btnLink = domain + "/"
	case "admin_create_user":
		subject = fmt.Sprintf("账户已创建 - %s", siteName)
		title = "账户已创建"
		username := html.EscapeString(data["username"])
		email := html.EscapeString(data["email"])
		password := html.EscapeString(data["password"])
		loginURL := domain + "/login"
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，管理员已为您创建了 %s 账户。</p>
<div style="background:#F9FAFB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">邮箱：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0 0 8px">用户名：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0 0 8px">初始密码：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0">登录地址：<a href="%s" style="color:#4F46E5">%s</a></p>
</div>
<p style="font-size:13px;color:#E11D48;margin:0">⚠️ 请登录后立即修改密码。</p>`, siteName, email, username, password, loginURL, loginURL)
		btnText = "立即登录"
		btnLink = loginURL
	// TEMPLATE_PLACEHOLDER_3
	case "account_disabled":
		subject = fmt.Sprintf("账户已被禁用 - %s", siteName)
		title = "账户已禁用"
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的 %s 账户已被管理员禁用。</p>
<div style="background:#FEF2F2;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#991B1B;margin:0">您的账户已无法登录和使用服务。如有疑问，请联系客服。</p>
</div>`, siteName)
	case "account_enabled":
		subject = fmt.Sprintf("账户已恢复 - %s", siteName)
		title = "账户已恢复"
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的 %s 账户已被管理员恢复启用。</p>
<div style="background:#F0FDF4;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#166534;margin:0">✅ 您现在可以正常登录和使用服务了。</p>
</div>`, siteName)
		btnText = "立即登录"
		btnLink = domain + "/login"
	case "account_deleted":
		subject = fmt.Sprintf("账户已删除 - %s", siteName)
		title = "账户已删除"
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的 %s 账户已被删除。</p>
<div style="background:#FEF2F2;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#991B1B;margin:0">您的所有数据（包括订阅、订单等）已被清除。如有疑问，请联系客服。</p>
</div>`, siteName)
	case "subscription_reset":
		subject = fmt.Sprintf("订阅地址已重置 - %s", siteName)
		title = "订阅地址已重置"
		resetBy := html.EscapeString(data["reset_by"])
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的订阅地址已被%s重置。</p>
<div style="background:#FFFBEB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#92400E;margin:0">⚠️ 旧的订阅地址已失效，所有已连接设备已被清除。请使用新的订阅地址重新配置客户端。</p>
</div>`, resetBy)
		btnText = "查看新订阅"
		btnLink = domain + "/subscription"
	case "abnormal_login":
		subject = fmt.Sprintf("异常登录提醒 - %s", siteName)
		title = "异常登录提醒"
		ip := html.EscapeString(data["ip"])
		location := html.EscapeString(data["location"])
		loginTime := html.EscapeString(data["time"])
		ua := html.EscapeString(data["user_agent"])
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的账户检测到一次异常登录。</p>
<div style="background:#FEF2F2;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">IP 地址：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0 0 8px">位置：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0 0 8px">时间：<strong>%s</strong></p>
  <p style="font-size:13px;color:#666;margin:0;word-break:break-all">设备：%s</p>
</div>
<p style="font-size:14px;color:#E11D48;margin:0">如果这不是您本人操作，请立即修改密码。</p>`, ip, location, loginTime, ua)
		btnText = "修改密码"
		btnLink = domain + "/settings"
	case "unpaid_order":
		subject = fmt.Sprintf("您有未支付的订单 - %s", siteName)
		title = "订单待支付提醒"
		orderNo := html.EscapeString(data["order_no"])
		packageName := html.EscapeString(data["package_name"])
		amount := html.EscapeString(data["amount"])
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您有一笔订单尚未支付。</p>
<div style="background:#FFFBEB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">订单号：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0 0 8px">套餐：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0">金额：<strong>¥%s</strong></p>
</div>
<p style="font-size:14px;color:#666;margin:0">订单将在 30 分钟后自动取消，请尽快完成支付。</p>`, orderNo, packageName, amount)
		btnText = "立即支付"
		btnLink = domain + "/orders"
	case "new_order":
		subject = fmt.Sprintf("新订单通知 - %s", siteName)
		title = "新订单"
		orderNo := html.EscapeString(data["order_no"])
		packageName := html.EscapeString(data["package_name"])
		amount := html.EscapeString(data["amount"])
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您已成功创建订单。</p>
<div style="background:#F9FAFB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">订单号：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0 0 8px">套餐：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0">金额：<strong>¥%s</strong></p>
</div>
<p style="font-size:14px;color:#666;margin:0">请尽快完成支付以激活服务。</p>`, orderNo, packageName, amount)
		btnText = "去支付"
		btnLink = domain + "/orders"
	default:
		subject = fmt.Sprintf("通知 - %s", siteName)
		title = "通知"
		content = `<p style="font-size:15px;color:#333">` + html.EscapeString(data["message"]) + `</p>`
	}

	htmlBody = buildEmailHTML(siteName, domain, title, content, btnText, btnLink)
	return
}

func buildEmailHTML(siteName, domain, title, content, btnText, btnLink string) string {
	btnHTML := ""
	if btnText != "" && btnLink != "" {
		btnHTML = fmt.Sprintf(`<div style="text-align:center;margin:28px 0 8px">
  <a href="%s" target="_blank" style="display:inline-block;background:linear-gradient(135deg,#6366F1,#4F46E5);color:#ffffff;text-decoration:none;padding:12px 32px;border-radius:6px;font-size:15px;font-weight:bold">%s</a>
</div>`, btnLink, btnText)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head>
<body style="margin:0;padding:0;background-color:#F3F4F6;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color:#F3F4F6">
<tr><td align="center" style="padding:32px 16px">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%%">
<tr><td style="background:linear-gradient(135deg,#6366F1,#4F46E5);padding:28px 32px;border-radius:12px 12px 0 0;text-align:center">
  <h1 style="margin:0;font-size:22px;color:#ffffff;font-weight:bold">%s</h1>
</td></tr>
<tr><td style="background:#ffffff;padding:32px;border-radius:0 0 12px 12px">
  <h2 style="margin:0 0 20px;font-size:18px;color:#1F2937;font-weight:600">%s</h2>
  %s
  %s
</td></tr>
<tr><td style="padding:24px 32px;text-align:center">
  <p style="margin:0 0 4px;font-size:12px;color:#9CA3AF">© %s. All rights reserved.</p>
  <p style="margin:0;font-size:12px;color:#9CA3AF">此邮件由系统自动发送，请勿直接回复。</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`, siteName, title, content, btnHTML, siteName)
}
