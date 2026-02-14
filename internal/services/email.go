package services

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
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
	db := database.GetDB()
	keys := []string{"smtp_host", "smtp_port", "smtp_username", "smtp_password", "smtp_from"}
	var configs []models.SystemConfig
	db.Where("`key` IN ?", keys).Find(&configs)

	m := make(map[string]string)
	for _, c := range configs {
		m[c.Key] = c.Value
	}

	host := m["smtp_host"]
	if host == "" {
		return nil, fmt.Errorf("SMTP 未配置: smtp_host 为空")
	}

	port := 587
	if p, err := strconv.Atoi(m["smtp_port"]); err == nil && p > 0 {
		port = p
	}

	return &SMTPConfig{
		Host:     host,
		Port:     port,
		Username: m["smtp_username"],
		Password: m["smtp_password"],
		From:     m["smtp_from"],
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
	from := cfg.From
	if from == "" {
		from = cfg.Username
	}

	msg := buildMIME(from, to, subject, body)
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
		if err = client.Mail(from); err != nil {
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
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
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
		} else {
			eq.Status = "sent"
			eq.SentAt = &now
			db.Save(eq)
		}
	}
}
