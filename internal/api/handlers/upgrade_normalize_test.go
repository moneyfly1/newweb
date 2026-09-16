package handlers

import "testing"

// 「只续期」与「按天续期」的回归。
//
// 客户端实测 bug：界面让用户选「增加天数」（add_days），而后端只认 extend_months，
// 于是金额不变、到期时间也不变 —— 用户花了钱没续上。这里把换算与校验钉住。
func TestNormalizeUpgradeRequest(t *testing.T) {
	cases := []struct {
		name             string
		devices, months  int
		days             int
		wantDev, wantMon int
		wantErr          bool
	}{
		{"只加设备", 1, 0, 0, 1, 0, false},
		{"只续期 1 个月", 0, 1, 0, 0, 1, false},
		{"只续期 30 天 = 1 个月", 0, 0, 30, 0, 1, false},
		{"只续期 90 天 = 3 个月", 0, 0, 90, 0, 3, false},
		{"只续期 365 天 = 13 个月（不夹取，上限 120）", 0, 0, 365, 0, 13, false},
		{"设备+续期", 2, 3, 0, 2, 3, false},
		{"两者都为 0 要报错", 0, 0, 0, 0, 0, true},
		{"负数设备要报错", -1, 0, 0, 0, 0, true},
		{"超过 100 台要报错", 101, 0, 0, 0, 0, true},
		{"月数超上限要报错", 1, 121, 0, 0, 0, true},
	}
	for _, c := range cases {
		dev, mon, err := normalizeUpgradeRequest(c.devices, c.months, c.days)
		if c.wantErr {
			if err == nil {
				t.Fatalf("%s: 期望报错，实际通过（dev=%d mon=%d）", c.name, dev, mon)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: 不该报错：%v", c.name, err)
		}
		if dev != c.wantDev || mon != c.wantMon {
			t.Fatalf("%s: 得到 dev=%d mon=%d，期望 dev=%d mon=%d", c.name, dev, mon, c.wantDev, c.wantMon)
		}
	}
}
