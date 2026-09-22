package services

import (
	"testing"
)

// 节点健康检查必须能解析订阅里所有支持的协议。
//
// 曾经只有 vmess/vless/trojan/ss 能取到 host:port，其它协议（hysteria2、socks5…）
// 一律返回 "unsupported protocol" → 被判定为离线。线上实测：hysteria2 节点
// 114 条全部显示离线、延迟 0，客户看到的就是「导入 socks/hysteria2 节点后一直超时」。
// 这些用例钉住「健康检查与订阅生成用的是同一套解析」。
func TestExtractHostPortSupportsAllProtocols(t *testing.T) {
	cases := []struct {
		name string
		link string
		want string
	}{
		{"socks5 明文账号密码", "socks://user:pass@new.miyaip.app:8001#节点", "new.miyaip.app:8001"},
		{"socks5 Base64 账号密码", "socks://dXNlcjpwYXNz@new.miyaip.app:8001#节点", "new.miyaip.app:8001"},
		{"socks5 无账号", "socks5://example.com:1080", "example.com:1080"},
		{"hysteria2", "hysteria2://pass@example.com:443?sni=example.com#节点", "example.com:443"},
		{"hy2 别名", "hy2://pass@example.com:8443#节点", "example.com:8443"},
		{"hysteria", "hysteria://example.com:36712?protocol=udp&auth=abc#节点", "example.com:36712"},
		{"tuic", "tuic://uuid:pass@example.com:443?congestion_control=bbr#节点", "example.com:443"},
		{"anytls", "anytls://pass@example.com:8443#节点", "example.com:8443"},
		{"trojan", "trojan://pass@example.com:443?sni=example.com#节点", "example.com:443"},
		{"vless", "vless://uuid@example.com:443?encryption=none&security=tls#节点", "example.com:443"},
		{"http 代理", "http://user:pass@example.com:8080", "example.com:8080"},
		{"ss", "ss://YWVzLTI1Ni1nY206cGFzcw==@example.com:8388#节点", "example.com:8388"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractHostPort(tc.link)
			if err != nil {
				t.Fatalf("应能解析出 host:port，实际报错: %v", err)
			}
			if got != tc.want {
				t.Errorf("host:port 不对: got %q want %q", got, tc.want)
			}
		})
	}
}

// 无法解析的输入必须报错（不能返回空地址让探测假装成功）
func TestExtractHostPortRejectsGarbage(t *testing.T) {
	for _, bad := range []string{"", "   ", "不是链接"} {
		if _, err := extractHostPort(bad); err == nil {
			t.Errorf("%q 应报错", bad)
		}
	}
}

// UDP 系协议只监听 UDP，用 TCP 探测必然失败——线上 114 条 hysteria2 节点就是这样
// 被全线判成离线、延迟 0 的。这里钉住「UDP 协议不做 TCP 探测」。
func TestUDPBasedProtocolsSkipTCPProbe(t *testing.T) {
	udpLinks := []string{
		"hysteria2://pass@example.com:443?sni=example.com#节点",
		"hy2://pass@example.com:8443#节点",
		"hysteria://example.com:36712?auth=abc#节点",
		"tuic://uuid:pass@example.com:443#节点",
		"wg://example.com:51820#节点",
	}
	for _, link := range udpLinks {
		if !isUDPBasedProtocol(link) {
			t.Errorf("%q 应识别为 UDP 系协议（不能走 TCP 探测）", link)
		}
	}
	tcpLinks := []string{
		"socks://user:pass@example.com:8001#节点",
		"socks5://example.com:1080",
		"vless://uuid@example.com:443?security=tls#节点",
		"trojan://pass@example.com:443#节点",
		"http://user:pass@example.com:8080",
	}
	for _, link := range tcpLinks {
		if isUDPBasedProtocol(link) {
			t.Errorf("%q 是 TCP 协议，应走 TCP 探测", link)
		}
	}
}
