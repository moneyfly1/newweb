package handlers

import (
	"testing"
	"time"
)

// 后台改「到期时间」曾经是**静默失败**：解析不出来就跳过、接口照样回成功，
// 面板显示已改好、数据库里还是旧值 —— 用户看到的就是
// 「我在管理员后台改了到期时间，客户端一直没变」。
// 这里钉住：该接受的格式要接受，不该接受的必须报错。
func TestParseExpireTimeParam(t *testing.T) {
	ok := []struct {
		name string
		in   interface{}
		want time.Time
	}{
		{"面板 toISOString()（带毫秒 Z）", "2029-01-01T00:00:00.000Z", time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"RFC3339 带时区", "2029-01-01T08:30:00+08:00", time.Date(2029, 1, 1, 8, 30, 0, 0, time.FixedZone("", 8*3600))},
		{"无时区的 T 分隔", "2029-01-01T00:00:00", time.Date(2029, 1, 1, 0, 0, 0, 0, time.Local)},
		{"空格分隔", "2029-01-01 12:00:00", time.Date(2029, 1, 1, 12, 0, 0, 0, time.Local)},
		{"只有日期", "2029-01-01", time.Date(2029, 1, 1, 0, 0, 0, 0, time.Local)},
		{"毫秒时间戳（字符串）", "1861920000000", time.UnixMilli(1861920000000)},
		{"秒时间戳（数字）", float64(1861920000), time.Unix(1861920000, 0)},
		{"time.Time 原样", time.Date(2030, 5, 6, 7, 8, 9, 0, time.UTC), time.Date(2030, 5, 6, 7, 8, 9, 0, time.UTC)},
	}
	for _, tc := range ok {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseExpireTimeParam(tc.in)
			if err != nil {
				t.Fatalf("应该能解析，却报错：%v", err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("解析结果不对：got %s want %s", got, tc.want)
			}
		})
	}

	bad := []struct {
		name string
		in   interface{}
	}{
		{"空字符串", ""},
		{"只有空白", "   "},
		{"中文日期", "2029年1月1日"},
		{"乱写", "tomorrow"},
		{"不支持的类型", []string{"2029-01-01"}},
	}
	for _, tc := range bad {
		t.Run("报错："+tc.name, func(t *testing.T) {
			if _, err := parseExpireTimeParam(tc.in); err == nil {
				t.Fatalf("这种输入必须报错（否则就是静默失败）")
			}
		})
	}
}
