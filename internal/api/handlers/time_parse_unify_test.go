package handlers

import (
	"testing"
	"time"
)

// 后台「到期时间」有两个入口（用户管理 admin_users、订阅管理 admin_subscriptions），
// 曾经各写一套解析：订阅管理不接受时间戳、日期回退布局按 UTC 解释。
// 现在两处都走 parseExpireTimeParam，这里钉住它们的输入集合完全一致。
func TestExpireTimeParsersAcceptSameInputs(t *testing.T) {
	inputs := []string{
		"2029-01-01T00:00:00.000Z",
		"2029-01-01T08:30:00+08:00",
		"2029-01-01T00:00:00",
		"2029-01-01 12:00:00",
		"2029-01-01",
		"1861920000000",
	}
	for _, in := range inputs {
		expire, errExpire := parseExpireTimeParam(in)
		coupon, errCoupon := parseCouponTime(in)
		if errExpire != nil || errCoupon != nil {
			t.Errorf("%q 必须在两个入口都能解析: expire=%v coupon=%v", in, errExpire, errCoupon)
			continue
		}
		if !expire.Equal(coupon) {
			t.Errorf("%q 在两个入口解析结果不同: expire=%s coupon=%s", in, expire, coupon)
		}
	}
}

// 日期-only 输入必须在两个入口得到同一时刻（此前一个 UTC 一个 Local，差 8 小时）。
func TestExpireTimeDateOnlyConsistentAcrossEntries(t *testing.T) {
	expire, err := parseExpireTimeParam("2029-01-01")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	coupon, err := parseCouponTime("2029-01-01")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if !expire.Equal(coupon) {
		t.Fatalf("同一日期在两个入口得到不同时刻: %s vs %s", expire, coupon)
	}
	if expire.Hour() != 0 || expire.Location() != time.Local {
		t.Errorf("日期-only 应按本地零点解释，实际 %s (%s)", expire, expire.Location())
	}
}

// 优惠券时间解析失败必须报错：以前是 `if err == nil`，失败就把原始字符串
// 写进 time 列，接口照样回成功（面板显示已保存、库里是脏值）。
func TestCouponTimeRejectsGarbage(t *testing.T) {
	for _, in := range []string{"", "  ", "明天", "2029年1月1日"} {
		if _, err := parseCouponTime(in); err == nil {
			t.Errorf("%q 必须报错", in)
		}
	}
}
