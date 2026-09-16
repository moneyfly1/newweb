package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseExpireTimeParam 解析后台传来的「到期时间」。
//
// 为什么要单独抽出来：以前两个后台更新接口（用户管理、订阅管理）都是
//
//	if t, err := time.Parse(time.RFC3339, s); err == nil { 写入 }
//
// ——**解析失败静默跳过**，接口照样回「成功」，面板上也显示成已经改好了，
// 数据库里却还是旧值。用户反馈「我在管理员后台改了到期时间，客户端没更新」
// 就是这类静默失败最典型的表现：客户端拿到的永远是旧值。
//
// 现在：解析失败**明确报错**（不再假装成功），并且顺手兼容面板可能发出的各种形态。
func parseExpireTimeParam(v interface{}) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case *time.Time:
		if t == nil {
			return time.Time{}, fmt.Errorf("到期时间不能为空")
		}
		return *t, nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return time.Time{}, fmt.Errorf("到期时间不能为空")
		}
		// 纯数字字符串 = 时间戳（前端 Date.getTime() 给的是毫秒）
		if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
			return expireTimeFromUnix(ts), nil
		}
		for _, layout := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
		} {
			if parsed, err := time.ParseInLocation(layout, s, time.Local); err == nil {
				return parsed, nil
			}
		}
		return time.Time{}, fmt.Errorf("到期时间格式不正确：%q（需要 2029-01-01T00:00:00+08:00 这样的格式）", s)
	case float64:
		return expireTimeFromUnix(int64(t)), nil
	case int64:
		return expireTimeFromUnix(t), nil
	case int:
		return expireTimeFromUnix(int64(t)), nil
	default:
		return time.Time{}, fmt.Errorf("到期时间格式不正确")
	}
}

// expireTimeFromUnix 自动分辨秒与毫秒时间戳（1e12 以上按毫秒）。
func expireTimeFromUnix(ts int64) time.Time {
	if ts >= 1_000_000_000_000 {
		return time.UnixMilli(ts)
	}
	return time.Unix(ts, 0)
}
