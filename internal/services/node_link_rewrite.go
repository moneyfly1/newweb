package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// 节点链接的「改地址/改端口」重写。
//
// 背景：专线节点/节点的下发**完全以 config（分享链接）为准**，
// 数据库里的 domain / port 只是给人看的派生字段。于是后台编辑表单里改「域名/端口」
// 会出现「列表变了、用户拿到的还是旧地址」的假生效。
// 这里提供反向操作：把链接里的 host:port 换成新值，让编辑真正生效。
//
// 设计原则：
//   - 只改 host 与 port，其余部分（UUID、密码、参数、备注名、base64 编码风格）原样保留
//   - 认不出的形态返回明确错误，由调用方提示管理员「请直接编辑配置链接」，绝不静默改坏

// RewriteNodeLinkHostPort 把节点链接里的服务器地址与端口替换为新值。
func RewriteNodeLinkHostPort(link string, newHost string, newPort int) (string, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", fmt.Errorf("配置链接为空")
	}
	if newPort <= 0 || newPort > 65535 {
		return "", fmt.Errorf("端口不合法: %d", newPort)
	}

	switch {
	case strings.HasPrefix(link, "vmess://"):
		// 只支持 base64(JSON) 形态：项目自身的解析器也只认这一种，
		// 改成 URL 形态会写出一条谁也解析不了的链接，所以直接报错让管理员手动改配置。
		return rewriteVmessLink(link, newHost, newPort)
	case strings.HasPrefix(link, "ssr://"):
		return rewriteSSRLink(link, newHost, newPort)
	case strings.HasPrefix(link, "ss://"):
		return rewriteShadowsocksLink(link, newHost, newPort)
	case strings.HasPrefix(link, "vless://"),
		strings.HasPrefix(link, "trojan://"),
		strings.HasPrefix(link, "hysteria://"),
		strings.HasPrefix(link, "hysteria2://"),
		strings.HasPrefix(link, "hy2://"),
		strings.HasPrefix(link, "tuic://"),
		strings.HasPrefix(link, "socks5://"),
		strings.HasPrefix(link, "socks://"),
		strings.HasPrefix(link, "http://"),
		strings.HasPrefix(link, "https://"),
		strings.HasPrefix(link, "naive://"),
		strings.HasPrefix(link, "naive+https://"),
		strings.HasPrefix(link, "anytls://"),
		strings.HasPrefix(link, "wg://"),
		strings.HasPrefix(link, "wireguard://"):
		return rewriteURLFormLink(link, newHost, newPort)
	}
	return "", fmt.Errorf("暂不支持自动改写该协议链接（%s），请直接编辑「配置」字段", firstScheme(link))
}

func firstScheme(link string) string {
	if i := strings.Index(link, "://"); i > 0 {
		return link[:i]
	}
	return "未知"
}

// rewriteURLFormLink 处理 scheme://[userinfo@]host:port[/path][?query][#fragment] 形态。
func rewriteURLFormLink(link string, newHost string, newPort int) (string, error) {
	schemeEnd := strings.Index(link, "://")
	if schemeEnd < 0 {
		return "", fmt.Errorf("链接缺少协议头")
	}
	scheme := link[:schemeEnd+3]
	rest := link[schemeEnd+3:]

	// 把 authority 与 path/query/fragment 分开
	authorityEnd := len(rest)
	for i, r := range rest {
		if r == '/' || r == '?' || r == '#' {
			authorityEnd = i
			break
		}
	}
	authority := rest[:authorityEnd]
	tail := rest[authorityEnd:]

	userinfo := ""
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		userinfo = authority[:at+1]
		authority = authority[at+1:]
	}

	if _, _, err := splitHostPortLoose(authority); err != nil {
		return "", err
	}
	host := newHost
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]" // IPv6
	}
	return scheme + userinfo + host + ":" + strconv.Itoa(newPort) + tail, nil
}

// splitHostPortLoose 解析 host:port（兼容 IPv6 方括号写法与缺省端口）。
func splitHostPortLoose(s string) (string, int, error) {
	if s == "" {
		return "", 0, fmt.Errorf("链接里没有服务器地址")
	}
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		return "", 0, fmt.Errorf("无法解析服务器地址: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("端口不是数字: %s", portStr)
	}
	return host, port, nil
}

// rewriteVmessLink 处理 base64(JSON) 形态的 vmess 链接。
func rewriteVmessLink(link string, newHost string, newPort int) (string, error) {
	body := strings.TrimPrefix(link, "vmess://")
	if i := strings.IndexAny(body, "?"); i >= 0 {
		body = body[:i] // 有些客户端会在 base64 后追加参数
	}
	raw, flavor, err := decodeBase64Any(body)
	if err != nil {
		return "", fmt.Errorf("vmess 链接不是合法 base64")
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", fmt.Errorf("vmess 链接不是 JSON 结构")
	}
	if _, ok := obj["add"]; !ok {
		if _, ok2 := obj["address"]; !ok2 {
			return "", fmt.Errorf("vmess JSON 里没有服务器地址字段")
		}
	}
	if _, ok := obj["add"]; ok {
		obj["add"] = newHost
	} else {
		obj["address"] = newHost
	}
	obj["port"] = strconv.Itoa(newPort)

	out, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return "vmess://" + encodeBase64Like(out, flavor), nil
}

// rewriteSSRLink 处理 ssr://base64(host:port:protocol:method:obfs:base64pass/?params)。
func rewriteSSRLink(link string, newHost string, newPort int) (string, error) {
	body := strings.TrimPrefix(link, "ssr://")
	if i := strings.Index(body, "?"); i >= 0 {
		body = body[:i]
	}
	raw, flavor, err := decodeBase64Any(body)
	if err != nil {
		return "", fmt.Errorf("ssr 链接不是合法 base64")
	}
	mainAndParams := strings.SplitN(string(raw), "/?", 2)
	parts := strings.SplitN(mainAndParams[0], ":", 6)
	if len(parts) < 6 {
		return "", fmt.Errorf("ssr 链接结构不完整")
	}
	parts[0] = newHost
	parts[1] = strconv.Itoa(newPort)
	rebuilt := strings.Join(parts, ":")
	if len(mainAndParams) == 2 {
		rebuilt += "/?" + mainAndParams[1]
	}
	return "ssr://" + encodeBase64Like([]byte(rebuilt), flavor), nil
}

// rewriteShadowsocksLink 处理两种 ss 形态：
//  1. SIP002: ss://base64(method:password)@host:port?params#name
//  2. 旧式:   ss://base64(method:password@host:port)#name
func rewriteShadowsocksLink(link string, newHost string, newPort int) (string, error) {
	rest := strings.TrimPrefix(link, "ss://")

	// 先尝试 SIP002（含 @ 且 @ 之前不是整段 base64 的旧式）
	if at := strings.Index(rest, "@"); at >= 0 {
		userinfo := rest[:at]
		after := rest[at+1:]
		authority := after
		tail := ""
		for i, r := range after {
			if r == '/' || r == '?' || r == '#' {
				authority = after[:i]
				tail = after[i:]
				break
			}
		}
		if _, _, err := splitHostPortLoose(authority); err == nil {
			host := newHost
			if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
				host = "[" + host + "]"
			}
			return "ss://" + userinfo + "@" + host + ":" + strconv.Itoa(newPort) + tail, nil
		}
	}

	// 旧式：整段 base64，解码后是 method:password@host:port
	body := rest
	frag := ""
	if i := strings.Index(body, "#"); i >= 0 {
		frag = body[i:]
		body = body[:i]
	}
	raw, flavor, err := decodeBase64Any(body)
	if err != nil {
		return "", fmt.Errorf("ss 链接不是合法 base64")
	}
	s := string(raw)
	at := strings.LastIndex(s, "@")
	if at < 0 {
		return "", fmt.Errorf("ss 链接里没有服务器地址")
	}
	creds := s[:at]
	authority := s[at+1:]
	if _, _, err := splitHostPortLoose(authority); err != nil {
		return "", err
	}
	host := newHost
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	rebuilt := creds + "@" + host + ":" + strconv.Itoa(newPort)
	return "ss://" + encodeBase64Like([]byte(rebuilt), flavor) + frag, nil
}

type b64Flavor int

const (
	b64Std b64Flavor = iota
	b64RawURL
	b64URL
)

// decodeBase64Any 依次尝试几种 base64 变体，返回解码结果与原始风格（用于编码回去时保持一致）。
func decodeBase64Any(s string) ([]byte, b64Flavor, error) {
	s = strings.TrimRight(strings.TrimSpace(s), "=")
	if raw, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return raw, b64Std, nil
	}
	if raw, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return raw, b64RawURL, nil
	}
	if raw, err := base64.URLEncoding.DecodeString(s); err == nil {
		return raw, b64URL, nil
	}
	return nil, b64Std, fmt.Errorf("base64 decode failed")
}

func encodeBase64Like(raw []byte, flavor b64Flavor) string {
	switch flavor {
	case b64RawURL:
		return base64.RawURLEncoding.EncodeToString(raw)
	case b64URL:
		return strings.TrimRight(base64.URLEncoding.EncodeToString(raw), "=")
	default:
		return strings.TrimRight(base64.StdEncoding.EncodeToString(raw), "=")
	}
}

// DetectNodeTypeFromLink 从链接判断协议类型（对内部实现的导出包装，
// 供后台编辑时按「配置链接」反推 protocol，避免配置与协议字段不一致）。
func DetectNodeTypeFromLink(link string) string {
	return detectNodeTypeFromLink(link)
}

// NodeLinkHostPort 解析链接里的服务器与端口（供编辑时派生 domain/port 用）。
func NodeLinkHostPort(link string) (host string, port int, err error) {
	h, p, err := ExtractDomainPortFromNodeLink(link)
	if err != nil {
		return "", 0, err
	}
	return h, p, nil
}

// EnsureURLHostPort 供测试与调用方构造合法 URL 形态链接时复用。
var _ = url.Parse
