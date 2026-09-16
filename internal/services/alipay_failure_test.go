package services

import (
	"strings"
	"testing"
)

// 支付宝当面付失败的两种真实成因（线上实测）必须翻译成用户能行动的一句话，
// 而不是把用户丢到一个必然报错的网页上。
//
// 事故背景：设备升级 ¥1747.66 的订单走支付宝时，当面付因**单笔限额**被拒，
// 后端静默降级到「电脑网站支付」——而该应用没签约这个产品，返回的网页打开就是
// 支付宝的 `insufficient-isv-permissions` 报错页。用户看到的是「点了支付宝没反应 / 报错」。
func TestAlipayFailureMessage(t *testing.T) {
	cases := []struct {
		name    string
		subCode string
		subMsg  string
		amount  string
		want    []string
	}{
		{
			name:    "单笔收款限额（实测 ¥1000）",
			subCode: "ACQ.BEYOND_PER_RECEIPT_SINGLE_RESTRICTION",
			amount:  "1747.66",
			want:    []string{"单笔限额", "1747.66"},
		},
		{
			name:    "应用未签约当面付",
			subCode: "insufficient-isv-permissions",
			amount:  "71.00",
			want:    []string{"未签约"},
		},
		{
			name:    "其它业务失败：带上支付宝原文",
			subCode: "ACQ.SYSTEM_ERROR",
			subMsg:  "系统繁忙，请稍后重试",
			amount:  "71.00",
			want:    []string{"系统繁忙"},
		},
		{
			name:   "网络/未知失败：给人话",
			amount: "71.00",
			want:   []string{"支付宝下单失败", "其它支付方式"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := alipayFailureMessage(tc.subCode, tc.subMsg, tc.amount)
			if strings.TrimSpace(got) == "" {
				t.Fatalf("失败原因不能为空（用户会看到空白错误）")
			}
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("message %q 应该包含 %q", got, w)
				}
			}
		})
	}
}

// 降级开关是唯一读库的地方（utils.GetSetting 需要已初始化的 DB），
// 这里只钉住「拿不到配置时不降级」的口径：GetSetting 在没有 DB 时会 panic，
// 所以生产路径上的默认值必须由调用方（AlipayCreateOrderEx）保证 —— 见
// `if !alipayAllowPagePay() { return nil, ... }`，默认分支就是拒绝降级。
func TestAlipayFailureMessageDoesNotLeakStack(t *testing.T) {
	got := alipayFailureMessage("", "", "71.00")
	if strings.Contains(got, "dial tcp") || strings.Contains(got, "EOF") {
		t.Errorf("不该把底层网络细节直接摊给用户: %q", got)
	}
	if !strings.Contains(got, "支付宝下单失败") {
		t.Errorf("要给出可读的失败说明: %q", got)
	}
}
