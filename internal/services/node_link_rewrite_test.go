package services

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

// 这些用例覆盖「后台改域名/端口 → 链接被正确改写 → 下发跟着变」的核心链路。
// 每种协议都用真实形态的样例，并断言：改完之后能重新解析出新的 host/port，
// 且其它关键字段（UUID、密码、参数、备注名）没被改坏。

func mustRewrite(t *testing.T, link, host string, port int) string {
	t.Helper()
	out, err := RewriteNodeLinkHostPort(link, host, port)
	if err != nil {
		t.Fatalf("rewrite failed: %v (link=%s)", err, link)
	}
	return out
}

func assertHostPort(t *testing.T, link, wantHost string, wantPort int) {
	t.Helper()
	host, port, err := NodeLinkHostPort(link)
	if err != nil {
		t.Fatalf("解析改写后的链接失败: %v (link=%s)", err, link)
	}
	if host != wantHost || port != wantPort {
		t.Fatalf("改写后 host/port 不符: got %s:%d, want %s:%d (link=%s)", host, port, wantHost, wantPort, link)
	}
}

func TestRewriteVmessBase64JSON(t *testing.T) {
	payload := map[string]interface{}{
		"v": "2", "ps": "香港01", "add": "old.example.com", "port": "443",
		"id": "11111111-2222-3333-4444-555555555555", "aid": "0", "net": "ws",
		"type": "none", "host": "cdn.example.com", "path": "/ws", "tls": "tls",
	}
	body, _ := json.Marshal(payload)
	link := "vmess://" + base64.StdEncoding.EncodeToString(body)

	out := mustRewrite(t, link, "new.example.com", 8443)
	assertHostPort(t, out, "new.example.com", 8443)

	decoded, _ := base64.StdEncoding.DecodeString(trimEq(out[len("vmess://"):]))
	var got map[string]interface{}
	if err := json.Unmarshal(decoded, &got); err != nil {
		t.Fatalf("改写后不是合法 JSON: %v", err)
	}
	if got["id"] != "11111111-2222-3333-4444-555555555555" || got["path"] != "/ws" || got["ps"] != "香港01" {
		t.Fatalf("改写破坏了其它字段: %v", got)
	}
}

func TestRewriteSSRLegacy(t *testing.T) {
	inner := "cn09.old.com:8217:origin:chacha20-ietf:http_simple:" +
		base64.RawURLEncoding.EncodeToString([]byte("passwd")) +
		"/?obfsparam=" + base64.RawURLEncoding.EncodeToString([]byte("obfs.example.com")) +
		"&remarks=" + base64.RawURLEncoding.EncodeToString([]byte("新加坡17"))
	link := "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(inner))

	out := mustRewrite(t, link, "cn01.new.com", 9300)
	assertHostPort(t, out, "cn01.new.com", 9300)

	body := out[len("ssr://"):]
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		raw, err = base64.StdEncoding.DecodeString(body)
	}
	if err != nil {
		t.Fatalf("改写后 base64 解不开: %v", err)
	}
	s := string(raw)
	if !contains(s, "origin:chacha20-ietf:http_simple") {
		t.Fatalf("改写破坏了加密/混淆参数: %s", s)
	}
	if !contains(s, "obfsparam=") || !contains(s, "remarks=") {
		t.Fatalf("改写丢了参数: %s", s)
	}
}

func TestRewriteShadowsocksSIP002(t *testing.T) {
	userinfo := base64.RawURLEncoding.EncodeToString([]byte("aes-256-gcm:pwd123"))
	link := "ss://" + userinfo + "@1.2.3.4:8388/?plugin=obfs-local%3Bobfs%3Dhttp#备注名"

	out := mustRewrite(t, link, "5.6.7.8", 9999)
	assertHostPort(t, out, "5.6.7.8", 9999)
	if !contains(out, "plugin=obfs-local") || !contains(out, "#备注名") {
		t.Fatalf("改写丢了参数或备注: %s", out)
	}
	if !contains(out, userinfo+"@") {
		t.Fatalf("改写破坏了密码段: %s", out)
	}
}

func TestRewriteShadowsocksLegacy(t *testing.T) {
	inner := "aes-128-gcm:pwd@9.9.9.9:1080"
	link := "ss://" + base64.RawURLEncoding.EncodeToString([]byte(inner)) + "#旧式"

	out := mustRewrite(t, link, "8.8.8.8", 1081)
	assertHostPort(t, out, "8.8.8.8", 1081)
	if !contains(out, "#旧式") {
		t.Fatalf("改写丢了备注: %s", out)
	}
}

func TestRewriteURLFormLinks(t *testing.T) {
	cases := []string{
		"vless://uuid-1234@old.com:443?encryption=none&security=tls&sni=old.com#VLESS-香港",
		"trojan://pwd@old.com:443?sni=old.com#Trojan",
		"hysteria2://pwd@old.com:8443?sni=old.com#HY2",
		"hy2://pwd@old.com:8443#HY2短写",
		"tuic://uuid:pwd@old.com:443?congestion_control=bbr#TUIC",
		"socks5://user:pass@old.com:1080#SOCKS",
		"anytls://pwd@old.com:8443#AnyTLS",
	}
	for _, link := range cases {
		out := mustRewrite(t, link, "new.com", 2087)
		assertHostPort(t, out, "new.com", 2087)
		scheme := link[:indexOf(link, "://")]
		if !hasPrefix(out, scheme+"://") {
			t.Fatalf("协议头被改坏: %s", out)
		}
		if !contains(out, "#") {
			t.Fatalf("备注名被丢掉: %s", out)
		}
	}
}

func TestRewriteKeepsQueryAndFragment(t *testing.T) {
	link := "vless://u@a.com:443?encryption=none&security=reality&pbk=KEY#%E9%A6%99%E6%B8%AF"
	out := mustRewrite(t, link, "b.com", 8443)
	for _, want := range []string{"encryption=none", "security=reality", "pbk=KEY", "#%E9%A6%99%E6%B8%AF"} {
		if !contains(out, want) {
			t.Fatalf("改写丢了 %s: %s", want, out)
		}
	}
}

func TestRewriteUnsupportedLinkFails(t *testing.T) {
	if _, err := RewriteNodeLinkHostPort("unknown://whatever", "a.com", 443); err == nil {
		t.Fatal("不支持的协议应当报错，而不是静默返回")
	}
	if _, err := RewriteNodeLinkHostPort("vless://u@a.com:443#x", "b.com", 0); err == nil {
		t.Fatal("端口非法应当报错")
	}
	if _, err := RewriteNodeLinkHostPort("", "b.com", 443); err == nil {
		t.Fatal("空链接应当报错")
	}
}

// ---- 小工具 ----

func trimEq(s string) string {
	for len(s) > 0 && s[len(s)-1] == '=' {
		s = s[:len(s)-1]
	}
	return s
}

func contains(s, sub string) bool {
	return indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
