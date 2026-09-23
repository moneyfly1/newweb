package services

import "testing"

// 备用订阅域名解析：后台输入既可能是逗号/换行分隔，也可能是 JSON 数组；
// 必须去重、规范化（补 https://、去尾斜杠），否则会给客户展示坏地址。
func TestParseMirrorDomains(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"逗号分隔", "new.moneyfly.top, dollarsfly.top", []string{"https://dollarsfly.top", "https://new.moneyfly.top"}},
		{"换行与重复", "https://a.com\nhttps://a.com\nb.com", []string{"https://a.com", "https://b.com"}},
		{"JSON 数组", `["a.com","https://b.com/"]`, []string{"https://a.com", "https://b.com"}},
		{"空值", "   ", nil},
	}
	for _, tc := range cases {
		got := ParseMirrorDomains(tc.raw)
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
				break
			}
		}
	}
}

// 订阅地址拼接：域名可以只填域名，也可以带 http(s):// 与尾斜杠，结果必须一致。
func TestSubscriptionEndpointURL(t *testing.T) {
	const token = "abc123"
	want := "https://sub.example.com/api/v1/client/subscribe?token=abc123&type=clash"
	for _, base := range []string{"sub.example.com", "https://sub.example.com", "https://sub.example.com/", "  sub.example.com/  "} {
		if got := SubscriptionEndpointURL(base, token, "clash"); got != want {
			t.Errorf("base=%q got %q want %q", base, got, want)
		}
	}
	if got := SubscriptionEndpointURL("", token, "clash"); got != "" {
		t.Errorf("空域名应返回空串，实际 %q", got)
	}
	if got := SubscriptionEndpointURL("sub.example.com", "", "clash"); got != "" {
		t.Errorf("空 token 应返回空串，实际 %q", got)
	}
}

// 域名体检：非法/空域名必须如实报错，不能假装通过。
func TestCheckDomainRejectsBadHost(t *testing.T) {
	res := CheckDomain("", "订阅域名")
	if res.DNSOK || res.HTTPSOK || res.APIReachable {
		t.Errorf("空域名不应通过体检: %+v", res)
	}
	res = CheckDomain("this-domain-should-not-exist-zzz.invalid", "订阅域名")
	if res.DNSOK {
		t.Errorf("不存在的域名不应解析成功: %+v", res)
	}
	if len(res.Problems) == 0 {
		t.Errorf("应给出问题说明")
	}
}
