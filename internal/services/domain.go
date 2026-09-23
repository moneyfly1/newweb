package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"cboard/v2/internal/utils"
)

// 域名分层：站点域名 与 订阅域名 分开。
//
// 背景（真实需求）：域名在某些地区被屏蔽后，客户打不开网站、也拉不到订阅。
// 常见做法是「网站一个域名、订阅一个域名」，并在被封时切换到备用域名。
// 这里提供三层设置：
//   site_url            —— 主站（打开网站、支付回调、邮件里的站点链接）
//   subscription_domain —— 订阅专用域名（优先生成所有订阅地址）
//   subscription_mirrors—— 备用订阅域名（同一 token 的镜像地址，主域名被封时客户可切换）
//   backup_site_url     —— 备用站点域名（网站打不开时的入口）
//
// 关键事实：订阅接口只校验 token、不校验 Host，因此同一 token 在任意绑定到本机的
// 域名下都能用 —— 换域名不需要迁移数据，客户也不需要更换 token 或重新导入。

const (
	settingSubscriptionDomain = "subscription_domain"
	settingSubscriptionMirror = "subscription_mirrors"
	settingBackupSiteURL      = "backup_site_url"
)

// SubscriptionDomainSetting 当前订阅域名配置
type SubscriptionDomainSetting struct {
	SiteURL            string   `json:"site_url"`
	SubscriptionDomain string   `json:"subscription_domain"`
	Mirrors            []string `json:"subscription_mirrors"`
	BackupSiteURL      string   `json:"backup_site_url"`
}

// GetSubscriptionDomainSetting 读取域名相关设置（去空白 + 规范化）
func GetSubscriptionDomainSetting() SubscriptionDomainSetting {
	m := utils.GetSecretSettings(settingSubscriptionDomain, settingSubscriptionMirror, settingBackupSiteURL)
	return SubscriptionDomainSetting{
		SiteURL:            GetSiteURL(),
		SubscriptionDomain: normalizePublicBaseURL(m[settingSubscriptionDomain]),
		Mirrors:            ParseMirrorDomains(m[settingSubscriptionMirror]),
		BackupSiteURL:      normalizePublicBaseURL(m[settingBackupSiteURL]),
	}
}

// ParseMirrorDomains 解析备用订阅域名：既支持后台逗号/换行分隔，也支持 JSON 数组
func ParseMirrorDomains(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []string
	if strings.HasPrefix(raw, "[") {
		var arr []string
		if json.Unmarshal([]byte(raw), &arr) == nil {
			items = arr
		}
	}
	if items == nil {
		items = strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t'
		})
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, it := range items {
		v := normalizePublicBaseURL(it)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// SubscriptionBaseURL 返回生成订阅地址时要用的基地址：优先订阅专用域名。
func SubscriptionBaseURL() string {
	setting := GetSubscriptionDomainSetting()
	if setting.SubscriptionDomain != "" {
		return setting.SubscriptionDomain
	}
	return setting.SiteURL
}

// SubscriptionEndpointURL 拼一条订阅地址
func SubscriptionEndpointURL(base, token, subType string) string {
	base = normalizePublicBaseURL(base)
	if base == "" || token == "" {
		return ""
	}
	out := base + "/api/v1/client/subscribe?token=" + url.QueryEscape(token)
	if subType != "" {
		out += "&type=" + url.QueryEscape(subType)
	}
	return out
}

// SubscriptionMirrorURL 返回同一订阅的镜像地址（含主地址、去重、顺序稳定）。
// 面板用它在主地址之外额外展示「备用订阅地址」，客户在主域名被封时可切换。
func SubscriptionMirrorURL(token, subType string) []map[string]string {
	if token == "" {
		return nil
	}
	setting := GetSubscriptionDomainSetting()
	type pair struct{ label, base string }
	candidates := []pair{}
	if setting.SubscriptionDomain != "" {
		candidates = append(candidates, pair{"订阅域名", setting.SubscriptionDomain})
	}
	for _, m := range setting.Mirrors {
		candidates = append(candidates, pair{"备用订阅域名", m})
	}
	if setting.SiteURL != "" {
		candidates = append(candidates, pair{"站点域名", setting.SiteURL})
	}

	seen := map[string]bool{}
	out := []map[string]string{}
	for _, c := range candidates {
		u := SubscriptionEndpointURL(c.base, token, subType)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, map[string]string{
			"label":  c.label,
			"domain": strings.TrimPrefix(strings.TrimPrefix(c.base, "https://"), "http://"),
			"url":    u,
		})
	}
	return out
}

// DomainCheckResult 单个域名的体检结果
type DomainCheckResult struct {
	Domain         string   `json:"domain"`
	Role           string   `json:"role"` // 站点 / 订阅 / 备用站点 / 备用订阅
	DNSOK          bool     `json:"dns_ok"`
	ResolvedIPs    []string `json:"resolved_ips"`
	HTTPSOK        bool     `json:"https_ok"`
	HTTPStatus     int      `json:"http_status"`
	CertOK         bool     `json:"cert_ok"`
	CertIssuer     string   `json:"cert_issuer"`
	CertExpiry     string   `json:"cert_expiry"`
	CertDaysLeft   int      `json:"cert_days_left"`
	APIReachable   bool     `json:"api_reachable"`
	HTTPRedirectOK bool     `json:"http_redirect_ok"`
	Problems       []string `json:"problems"`
}

// CheckDomain 体检一个域名：DNS → TLS 证书 → 应用可达 → HTTP 跳转。
//
// 为什么要在应用里做：这几个环节任何一个没配好（解析没生效、证书没签发、
// 证书域名不匹配、后端没起来），客户看到的就是「打不开」或「证书错误」，
// 而在后台点一下就能定位到具体环节，不用登服务器逐个 curl。
func CheckDomain(host, role string) DomainCheckResult {
	host = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://"))
	host = strings.TrimSuffix(host, "/")
	res := DomainCheckResult{Domain: host, Role: role, ResolvedIPs: []string{}, Problems: []string{}}
	if host == "" {
		return res
	}

	// 1) DNS
	if ips, err := net.LookupHost(host); err == nil && len(ips) > 0 {
		res.DNSOK = true
		res.ResolvedIPs = ips
	} else {
		res.Problems = append(res.Problems, "DNS 解析失败：域名还没解析到服务器，或解析未生效")
		return res
	}

	// 2) TLS 证书（顺便拿到到期时间）
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 8 * time.Second}, "tcp", net.JoinHostPort(host, "443"),
		&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err != nil {
		res.Problems = append(res.Problems, "HTTPS 连接失败："+err.Error()+"（证书可能未签发或 443 未监听）")
		return res
	}
	state := conn.ConnectionState()
	_ = conn.Close()
	res.HTTPSOK = true
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		res.CertOK = true
		res.CertIssuer = cert.Issuer.CommonName
		res.CertExpiry = cert.NotAfter.Format(utils.LayoutDateTime)
		res.CertDaysLeft = int(time.Until(cert.NotAfter).Hours() / 24)
		if host != "" && cert.VerifyHostname(host) != nil {
			res.CertOK = false
			res.Problems = append(res.Problems, "证书域名与该域名不匹配（浏览器会直接报警告）")
		}
		if res.CertDaysLeft <= 0 {
			res.CertOK = false
			res.Problems = append(res.Problems, "证书已过期")
		} else if res.CertDaysLeft <= 7 {
			res.Problems = append(res.Problems, fmt.Sprintf("证书 %d 天后到期，请确认自动续期是否正常", res.CertDaysLeft))
		}
	}

	// 3) 应用是否真的在响应（后端 + nginx 都通）
	client := &http.Client{Timeout: 10 * time.Second}
	if resp, err := client.Get("https://" + host + "/api/v1/config"); err == nil {
		res.HTTPStatus = resp.StatusCode
		res.APIReachable = resp.StatusCode == http.StatusOK
		_ = resp.Body.Close()
	} else {
		res.Problems = append(res.Problems, "接口不可达："+err.Error())
	}
	if res.HTTPStatus != 0 && res.HTTPStatus != http.StatusOK {
		res.Problems = append(res.Problems, fmt.Sprintf("接口返回 %d（应为 200），检查 nginx 反代与后端服务", res.HTTPStatus))
	}
	if !res.APIReachable && res.HTTPSOK {
		// 首页可达但接口不通，常见于 nginx 反代段没加
		res.Problems = append(res.Problems, "HTTPS 能连但接口不可用：检查该域名的 nginx 配置是否包含 /api/ 反代")
	}

	// 4) HTTP 是否跳 HTTPS（不跳的话登录/订阅会有混合内容问题）
	noRedirect := &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	if resp, err := noRedirect.Get("http://" + host + "/"); err == nil {
		_ = resp.Body.Close()
		res.HTTPRedirectOK = resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound
		if !res.HTTPRedirectOK {
			res.Problems = append(res.Problems, "HTTP 没有跳转到 HTTPS")
		}
	}

	if len(res.Problems) == 0 {
		res.Problems = []string{}
	}
	return res
}

// CheckAllDomains 体检所有配置中的域名（站点 + 订阅 + 备用），按角色排序
func CheckAllDomains() []DomainCheckResult {
	setting := GetSubscriptionDomainSetting()
	type item struct{ host, role string }
	items := []item{}
	add := func(u, role string) {
		if u == "" {
			return
		}
		host := strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
		host = strings.TrimSuffix(host, "/")
		if host != "" {
			items = append(items, item{host, role})
		}
	}
	add(setting.SubscriptionDomain, "订阅域名")
	for _, m := range setting.Mirrors {
		add(m, "备用订阅域名")
	}
	add(setting.SiteURL, "站点域名")
	add(setting.BackupSiteURL, "备用站点域名")

	seen := map[string]bool{}
	out := []DomainCheckResult{}
	for _, it := range items {
		if seen[it.host] {
			continue
		}
		seen[it.host] = true
		out = append(out, CheckDomain(it.host, it.role))
	}
	return out
}

// NormalizeDomainInput 规范化后台输入的域名（允许只填域名或带 http(s):// 与路径）
func NormalizeDomainInput(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return normalizePublicBaseURL(raw)
}

// MirrorsToSetting 把多个备用域名序列化成设置值（逗号分隔，便于后台直接编辑）
func MirrorsToSetting(mirrors []string) string {
	return strings.Join(mirrors, ",")
}
