package handlers

import (
	"strings"
)

// 订阅访问控制：只有**软件客户端**能拿到配置。
//
// 背景（对齐 myweb 的 subscription_access）：
// 订阅地址一旦泄露，脚本（curl / python-requests / Go-http-client）、
// 在线转换站（subconverter / sub-web）、社交软件的链接预览爬虫
// （TelegramBot / WhatsApp / facebookexternalhit）都会来拉取配置——
// 它们既不是用户设备（不该登记设备、白占名额），也不该拿到完整节点。
//
// 判定顺序（与 myweb 一致）：
//  1. UA 本身就是链接（/subscriptions/clash/<token> 被当成 UA）→ 拦截；
//  2. 命中客户端标记 → 放行；
//  3. 空 UA → 拦截（正常客户端都会带 UA）；
//  4. 命中浏览器标记 → 放行（交给上层按"浏览器访问"处理：可看节点但不登记设备）；
//  5. 命中预览/自动化标记 → 拦截；
//  6. 其它未知 UA → 放行（不误伤小众客户端）。

// isScriptOrPreviewUA 判断 UA 是否属于「脚本 / 抓取 / 预览」——
// 这类请求直接返回空内容，既不下发配置也不登记设备。
func isScriptOrPreviewUA(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))

	// 1) UA 本身就是订阅链接（含 clash 字样，必须最先判断）
	if strings.HasPrefix(ua, "http://") || strings.HasPrefix(ua, "https://") ||
		strings.HasPrefix(ua, "vmess://") || strings.HasPrefix(ua, "vless://") ||
		strings.HasPrefix(ua, "ss://") || strings.HasPrefix(ua, "ssr://") ||
		strings.HasPrefix(ua, "trojan://") {
		return true
	}

	// 2) 客户端标记 → 放行
	clientMarkers := []string{
		"clash", "stash", "surge", "quantumult", "loon", "sing-box", "singbox",
		"shadowrocket", "v2ray", "v2rayn", "v2rayng", "hiddify", "nekobox",
		"surfboard", "karing", "openclash", "passwall", "moneyfly", "mbox",
		"mclash", "flclash", "mihomo", "verge", "nekoray", "sfa/", "sfi/", "sfm/",
	}
	for _, marker := range clientMarkers {
		if strings.Contains(ua, marker) {
			return false
		}
	}

	// 3) 空 UA → 拦截
	if ua == "" {
		return true
	}

	// 4) 浏览器标记 → 放行（浏览器访问有独立处理：可看节点但不登记设备）
	browserMarkers := []string{
		"chrome/", "crios/", "firefox/", "fxios/", "safari/", "edg/", "edge/",
		"opr/", "opera/", "msie", "trident/", "duckduckgo/", "vivaldi/",
	}
	for _, marker := range browserMarkers {
		if strings.Contains(ua, marker) {
			return false
		}
	}
	if strings.Contains(ua, "mozilla/") && strings.Contains(ua, "applewebkit/") {
		return false
	}

	// 5) 链接预览爬虫
	previewMarkers := []string{
		"whatsapp/", "facebookexternalhit", "facebot", "twitterbot", "telegrambot",
		"googlemessages", "dingtalkbot", "wxwork", "micromessenger", "slackbot",
		"discordbot", "linkedinbot", "pinterest", "redditbot", "bingbot",
		"googlebot", "applebot", "vkshare", "skypeuripreview", "zalo", "kakaotalk",
		"viber", "lark", "feishu", "mattermost", "rocket.chat",
	}
	for _, marker := range previewMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}

	// 6) 自动化 / 脚本 / 转换站
	automationMarkers := []string{
		"go-http-client", "curl/", "wget/", "python-requests", "python-urllib",
		"okhttp", "postmanruntime", "axios/", "node-fetch", "apache-httpclient",
		"libwww-perl", "got (", "got/", "undici", "java/", "java-http-client",
		"subconverter", "clash-subconverter", "sub-web", "node ", "go-resty",
		"insomnia", "httpie", "powershell", "winhttp", "libcurl",
	}
	for _, marker := range automationMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}

	return false
}
