package middleware

import (
	"testing"
	"time"
)

// Redis 未启用时用的是进程内存限流器，后台「解封某个 IP」必须也能清掉这份计数，
// 否则会出现「数据库里解封了、请求还是 429」。
func TestResetMemoryLimiters(t *testing.T) {
	rl := newMemRateLimiter(2, time.Minute)
	if !rl.allow("1.2.3.4") || !rl.allow("1.2.3.4") {
		t.Fatal("前两次应放行")
	}
	if rl.allow("1.2.3.4") {
		t.Fatal("第三次应被限流")
	}
	if !rl.allow("5.6.7.8") {
		t.Fatal("其它 IP 不应受影响")
	}

	// 列表里应能看到被限的 IP
	entries := ListMemoryLimiters()
	found := false
	for _, e := range entries {
		if e.IP == "1.2.3.4" {
			found = true
			if e.Count < 2 || e.Limit != 2 {
				t.Errorf("限流条目内容不对: %+v", e)
			}
		}
	}
	if !found {
		t.Fatalf("列表应包含被限流的 IP，实际: %+v", entries)
	}

	if cleared := ResetMemoryLimiters("1.2.3.4"); cleared == 0 {
		t.Fatal("应清理到该 IP 的计数")
	}
	if !rl.allow("1.2.3.4") {
		t.Error("解封后该 IP 应立即放行")
	}
	// 其它 IP 的计数不应被误清：达到上限后仍应被限流
	rl.allow("5.6.7.8") // 补到上限（limit=2）
	if rl.allow("5.6.7.8") {
		t.Error("其它 IP 达到上限后仍应被限流（不应被连带清理）")
	}
}

func TestSplitRateLimitKey(t *testing.T) {
	cases := []struct{ key, path, ip string }{
		{"ratelimit:/api/v1/auth/login:1.2.3.4", "/api/v1/auth/login", "1.2.3.4"},
		{"ratelimit:/api/v1/client/subscribe:240e:33d::1", "/api/v1/client/subscribe", "240e:33d::1"},
		{"ratelimit:noip", "noip", ""},
	}
	for _, tc := range cases {
		path, ip := splitRateLimitKey(tc.key)
		if path != tc.path || ip != tc.ip {
			t.Errorf("%q → (%q, %q)，期望 (%q, %q)", tc.key, path, ip, tc.path, tc.ip)
		}
	}
}
