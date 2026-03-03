package services

import (
	"fmt"
	"strings"
	"time"
)

// NotifyTemplate 通知模板结构
type NotifyTemplate struct {
	Emoji   string
	Title   string
	Fields  []NotifyField
	Footer  string
}

// NotifyField 通知字段
type NotifyField struct {
	Emoji string
	Label string
	Key   string // data map 中的键名
}

// GetNotifyTemplate 获取通知模板
func GetNotifyTemplate(eventType string) *NotifyTemplate {
	templates := map[string]*NotifyTemplate{
		"new_order": {
			Emoji: "📦",
			Title: "新订单",
			Fields: []NotifyField{
				{"🆔", "订单号", "order_no"},
				{"👤", "用户", "username"},
				{"📦", "套餐", "package_name"},
				{"💰", "金额", "amount"},
			},
		},
		"payment_success": {
			Emoji: "🎉",
			Title: "支付成功",
			Fields: []NotifyField{
				{"🆔", "订单号", "order_no"},
				{"👤", "用户", "username"},
				{"📦", "套餐", "package_name"},
				{"💰", "金额", "amount"},
			},
			Footer: "✅ 订单已自动处理\n📦 订阅已激活",
		},
		"recharge_success": {
			Emoji: "💰",
			Title: "充值成功",
			Fields: []NotifyField{
				{"🆔", "充值单号", "order_no"},
				{"👤", "用户", "username"},
				{"💰", "金额", "amount"},
			},
		},
		"new_ticket": {
			Emoji: "🎫",
			Title: "新工单",
			Fields: []NotifyField{
				{"🆔", "工单号", "ticket_no"},
				{"👤", "用户", "username"},
				{"📝", "标题", "title"},
			},
		},
		"new_user": {
			Emoji: "👋",
			Title: "新用户注册",
			Fields: []NotifyField{
				{"👤", "用户名", "username"},
				{"📧", "邮箱", "email"},
			},
			Footer: "✅ 已自动创建默认订阅",
		},
		"admin_create_user": {
			Emoji: "📋",
			Title: "管理员创建用户",
			Fields: []NotifyField{
				{"👤", "用户名", "username"},
				{"📧", "邮箱", "email"},
			},
		},
		"subscription_reset": {
			Emoji: "🔄",
			Title: "订阅重置",
			Fields: []NotifyField{
				{"👤", "用户", "username"},
				{"🔧", "操作者", "reset_by"},
			},
			Footer: "⚠️ 旧地址已失效",
		},
		"abnormal_login": {
			Emoji: "⚠️",
			Title: "异常登录",
			Fields: []NotifyField{
				{"👤", "用户", "username"},
				{"🌐", "IP", "ip"},
				{"📍", "位置", "location"},
			},
		},
		"unpaid_order": {
			Emoji: "⏳",
			Title: "未支付订单",
			Fields: []NotifyField{
				{"🆔", "订单号", "order_no"},
				{"👤", "用户", "username"},
				{"💰", "金额", "amount"},
			},
		},
		"expiry_reminder": {
			Emoji: "⏰",
			Title: "订阅到期提醒",
			Fields: []NotifyField{
				{"👤", "用户", "username"},
				{"⏰", "到期时间", "expire_time"},
			},
		},
	}

	return templates[eventType]
}

// RenderTelegramMessage 渲染 Telegram 消息（HTML 格式）
func RenderTelegramMessage(template *NotifyTemplate, data map[string]string) string {
	var sb strings.Builder

	// 标题
	sb.WriteString(fmt.Sprintf("%s <b>%s</b>\n\n", template.Emoji, template.Title))

	// 装饰线
	sb.WriteString("┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n")
	sb.WriteString(fmt.Sprintf("┃  📋 <b>%s详情</b>\n", template.Title))
	sb.WriteString("┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n")

	// 字段
	for _, field := range template.Fields {
		value := data[field.Key]
		if value == "" {
			value = "-"
		}
		// 金额字段添加货币符号
		if field.Key == "amount" && value != "-" {
			value = "¥" + value
		}
		sb.WriteString(fmt.Sprintf("%s <b>%s</b>: <code>%s</code>\n", field.Emoji, field.Label, value))
	}

	// 时间
	now := time.Now().Format("2006-01-02 15:04:05")
	sb.WriteString(fmt.Sprintf("🕐 <b>时间</b>: %s\n", now))

	// 页脚
	if template.Footer != "" {
		sb.WriteString("\n" + template.Footer)
	}

	return sb.String()
}

// RenderBarkMessage 渲染 Bark 消息（纯文本格式）
func RenderBarkMessage(template *NotifyTemplate, data map[string]string) string {
	var sb strings.Builder

	// 标题
	sb.WriteString(fmt.Sprintf("%s %s\n\n", template.Emoji, template.Title))

	// 字段
	for _, field := range template.Fields {
		value := data[field.Key]
		if value == "" {
			value = "-"
		}
		// 金额字段添加货币符号
		if field.Key == "amount" && value != "-" {
			value = "¥" + value
		}
		sb.WriteString(fmt.Sprintf("%s %s: %s\n", field.Emoji, field.Label, value))
	}

	// 时间
	now := time.Now().Format("2006-01-02 15:04:05")
	sb.WriteString(fmt.Sprintf("🕐 时间: %s\n", now))

	// 页脚
	if template.Footer != "" {
		sb.WriteString("\n" + template.Footer)
	}

	return sb.String()
}

// RenderNotifyTitle 渲染通知标题
func RenderNotifyTitle(siteName string, template *NotifyTemplate) string {
	return fmt.Sprintf("[%s] %s %s", siteName, template.Emoji, template.Title)
}
