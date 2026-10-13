package services

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRealityShortIDFix(t *testing.T) {
	cases := []struct {
		name    string
		link    string
		want    string // 期望的 short-id 值，"" 表示不应有 short-id 字段
		wantErr bool   // true 表示该节点应因非法 short-id 被剔除（返回 error）
	}{
		{"空 sid", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sni=example.com&type=tcp&flow=xtls-rprx-vision", "", false},
		{"合法 sid", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=a1b2c3d4&sni=example.com", "a1b2c3d4", false},
		{"大写 hex sid", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=A1B2C3D4&sni=example.com", "A1B2C3D4", false},
		{"科学计数法外观的合法 sid", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=450259e7&sni=example.com", "450259e7", false},
		{"非法 sid 含字母g", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=a1b2g3d4&sni=example.com", "", true},
		{"非法 sid 含小数点(客户端回写的浮点)", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=4502590000000.0&sni=example.com", "", true},
		{"非法 sid 奇数长度", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=abc&sni=example.com", "", true},
		{"非法 sid 超长(>16)", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=1234567890abcdef1&sni=example.com", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := VlessLinkToClashMap(tc.link, "test")
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望该节点因非法 short-id 被剔除，但解析成功了: %v", m)
				}
				return
			}
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			raw, _ := json.Marshal(m)
			got := ""
			if opts, ok := m["reality-opts"].(map[string]interface{}); ok {
				if v, ok := opts["short-id"]; ok {
					got = v.(string)
				}
			}
			if got != tc.want {
				t.Errorf("short-id = %q, want %q (config: %s)", got, tc.want, string(raw))
			}
		})
	}
}

// 回归测试：形如科学计数法的十六进制串不能被 YAML 当数值读走
func TestEscapeYAMLQuotesNumberLike(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"450259e7", `"450259e7"`}, // 核心回归用例
		{"4502590000000.0", `"4502590000000.0"`},
		{"443", `"443"`},
		{"true", `"true"`},
		{"null", `"null"`},
		{"2026-10-27", `"2026-10-27"`},
		{"a1b2c3d4", "a1b2c3d4"},
		{"www.apple.com", "www.apple.com"},
		{"", `""`},
	}
	for _, tc := range cases {
		if got := escapeYAML(tc.in); got != tc.want {
			t.Errorf("escapeYAML(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	for _, k := range []string{"short-id", "public-key", "uuid", "password", "servername"} {
		if !forceQuoteYAMLKey(k) {
			t.Errorf("forceQuoteYAMLKey(%q) = false, want true", k)
		}
	}
	// short-id 即使值本身不像数字也必须带引号输出
	var sb strings.Builder
	writeYAMLInlineValueForKey(&sb, "short-id", "a1b2c3d4")
	if sb.String() != `"a1b2c3d4"` {
		t.Errorf("short-id 未强制加引号: %s", sb.String())
	}
}
