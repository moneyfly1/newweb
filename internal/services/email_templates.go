package services

import (
	"fmt"
	"strings"

	"cboard/v2/internal/utils"
)

// RenderEmail renders a named email template with the given data.
// Returns the subject line and full HTML body.
func RenderEmail(templateName string, data map[string]string) (subject, htmlBody string) {
	settings := utils.GetSettings("site_name", "site_url", "domain_name")
	siteName := settings["site_name"]
	if siteName == "" {
		siteName = "CBoard"
	}
	domain := settings["site_url"]
	if domain == "" {
		domain = settings["domain_name"]
	}
	if domain != "" && !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}
	domain = strings.TrimRight(domain, "/")

	var title, content, btnText, btnLink string

	switch templateName {
	case "verification":
		subject = fmt.Sprintf("邮箱验证码 - %s", siteName)
		title = "邮箱验证码"
		code := data["code"]
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您正在进行邮箱验证操作。</p>
<div style="text-align:center;margin:24px 0">
  <span style="display:inline-block;font-size:32px;font-weight:bold;letter-spacing:8px;color:#4F46E5;background:#F0EFFF;padding:12px 24px;border-radius:8px">%s</span>
</div>
<p style="font-size:14px;color:#666;margin:0">验证码有效期 5 分钟，请勿泄露给他人。</p>`, code)
	case "reset_password":
		subject = fmt.Sprintf("密码重置 - %s", siteName)
		title = "密码重置"
		code := data["code"]
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您正在进行密码重置操作。</p>
<div style="text-align:center;margin:24px 0">
  <span style="display:inline-block;font-size:32px;font-weight:bold;letter-spacing:8px;color:#4F46E5;background:#F0EFFF;padding:12px 24px;border-radius:8px">%s</span>
</div>
<p style="font-size:14px;color:#666;margin:0">验证码有效期 15 分钟。如果这不是您的操作，请忽略此邮件。</p>`, code)

	case "welcome":
		subject = fmt.Sprintf("欢迎加入 %s", siteName)
		title = "欢迎加入"
		username := data["username"]
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">Hi %s，欢迎加入 %s！</p>
<p style="font-size:14px;color:#666;margin:0 0 8px">您的账户已创建成功，现在可以开始使用我们的服务了。</p>
<p style="font-size:14px;color:#666;margin:0">如有任何问题，请随时联系客服。</p>`, username, siteName)
		btnText = "开始使用"
		btnLink = domain + "/dashboard"

	case "subscription":
		subject = fmt.Sprintf("您的订阅信息 - %s", siteName)
		title = "订阅信息"
		clashURL := data["clash_url"]
		universalURL := data["universal_url"]
		expireTime := data["expire_time"]
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
		orderNo := data["order_no"]
		amount := data["amount"]
		packageName := data["package_name"]
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的订单已支付成功！</p>
<div style="background:#F0FDF4;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">订单号：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0 0 8px">套餐：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0">金额：<strong>¥%s</strong></p>
</div>
<p style="font-size:14px;color:#666;margin:0">订阅已自动激活，感谢您的支持！</p>`, orderNo, packageName, amount)
		btnText = "查看订单"
		btnLink = domain + "/orders"

	case "recharge_success":
		subject = fmt.Sprintf("充值成功 - %s", siteName)
		title = "充值成功"
		orderNo := data["order_no"]
		amount := data["amount"]
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的充值已成功到账！</p>
<div style="background:#F0FDF4;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">充值单号：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0">充值金额：<strong>¥%s</strong></p>
</div>
<p style="font-size:14px;color:#666;margin:0">余额已自动到账，感谢您的支持！</p>`, orderNo, amount)
		btnText = "查看余额"
		btnLink = domain + "/"

	case "expiry_reminder":
		days := data["days"]
		expireTime := data["expire_time"]
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
		expireTime := data["expire_time"]
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
		username := data["username"]
		password := data["password"]
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，管理员已为您创建了 %s 账户。</p>
<div style="background:#F9FAFB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">用户名：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0">初始密码：<strong>%s</strong></p>
</div>
<p style="font-size:13px;color:#E11D48;margin:0">⚠️ 请登录后立即修改密码。</p>`, siteName, username, password)
		btnText = "立即登录"
		btnLink = domain + "/login"

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
		resetBy := data["reset_by"]
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您的订阅地址已被%s重置。</p>
<div style="background:#FFFBEB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#92400E;margin:0">⚠️ 旧的订阅地址已失效，所有已连接设备已被清除。请使用新的订阅地址重新配置客户端。</p>
</div>`, resetBy)
		btnText = "查看新订阅"
		btnLink = domain + "/subscription"

	case "abnormal_login":
		subject = fmt.Sprintf("异常登录提醒 - %s", siteName)
		title = "异常登录提醒"
		ip := data["ip"]
		location := data["location"]
		loginTime := data["time"]
		ua := data["user_agent"]
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
		orderNo := data["order_no"]
		packageName := data["package_name"]
		amount := data["amount"]
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
		orderNo := data["order_no"]
		packageName := data["package_name"]
		amount := data["amount"]
		content = fmt.Sprintf(`<p style="font-size:15px;color:#333;margin:0 0 16px">您好，您已成功创建订单。</p>
<div style="background:#F9FAFB;border-radius:8px;padding:16px;margin:0 0 16px">
  <p style="font-size:14px;color:#333;margin:0 0 8px">订单号：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0 0 8px">套餐：<strong>%s</strong></p>
  <p style="font-size:14px;color:#333;margin:0">金额：<strong>¥%s</strong></p>
</div>
<p style="font-size:14px;color:#666;margin:0">请尽快完成支付以激活服务。</p>`, orderNo, packageName, amount)
		btnText = "去支付"
		btnLink = domain + "/orders"

	// PLACEHOLDER_TEMPLATES2
	default:
		subject = fmt.Sprintf("通知 - %s", siteName)
		title = "通知"
		content = `<p style="font-size:15px;color:#333">` + data["message"] + `</p>`
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
<!-- Header -->
<tr><td style="background:linear-gradient(135deg,#6366F1,#4F46E5);padding:28px 32px;border-radius:12px 12px 0 0;text-align:center">
  <h1 style="margin:0;font-size:22px;color:#ffffff;font-weight:bold">%s</h1>
</td></tr>
<!-- Body -->
<tr><td style="background:#ffffff;padding:32px;border-radius:0 0 12px 12px">
  <h2 style="margin:0 0 20px;font-size:18px;color:#1F2937;font-weight:600">%s</h2>
  %s
  %s
</td></tr>
<!-- Footer -->
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
