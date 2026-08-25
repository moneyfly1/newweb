package services

import (
	"encoding/json"
	"testing"
)

func TestRealityShortIDFix(t *testing.T) {
	cases := []struct {
		name string
		link string
		want string // 期望的 short-id 值，"" 表示不应有 short-id 字段
	}{
		{"空 sid", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sni=example.com&type=tcp&flow=xtls-rprx-vision", ""},
		{"合法 sid", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=a1b2c3d4&sni=example.com", "a1b2c3d4"},
		{"大写 hex sid", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=A1B2C3D4&sni=example.com", "A1B2C3D4"},
		{"非法 sid 含字母g", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=a1b2g3d4&sni=example.com", ""},
		{"非法 sid 含特殊字符", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=abc!@#&sni=example.com", ""},
		{"超长 sid", "vless://uuid@1.2.3.4:443?security=reality&pbk=abc123&sid=1234567890abcdef1234567890abcdef1&sni=example.com", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := VlessLinkToClashMap(tc.link, "test")
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
