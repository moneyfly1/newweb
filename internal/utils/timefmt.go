package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// 时间格式与解析的统一入口。
//
// 为什么集中：同一个「时间」在项目里此前有三套写法——
//   - 布局字符串到处手写（"2006-01-02 15:04:05" 出现 30 次、"2006-01-02" 47 次），
//     打错一个字符编译器不会报错，只会静默格式化出错误结果；
//   - 后台「到期时间」有 4 份各自的解析实现（用户管理、订阅管理、优惠券、CSV 导入），
//     接受的格式各不相同，同样填「2029-01-01」，有的接口能用时间戳、
//     有的接口直接报错；日期回退布局一个按 UTC 解释、一个按本地时区解释，
//     写入数据库的到期时刻能差 8 小时；
//   - CSV 导入解析失败时静默改用「默认 1 年」，导入结果和用户看到的表格对不上。
//
// 现在统一走这里：布局是常量，解析只有一个实现，失败一律明确报错。
const (
	// LayoutDate 仅日期，如 2029-01-01
	LayoutDate = "2006-01-02"
	// LayoutDateTime 日期+时间（秒），如 2029-01-01 12:00:00
	LayoutDateTime = "2006-01-02 15:04:05"
	// LayoutDateTimeShort 日期+时间（分钟），如 2029-01-01 12:00
	LayoutDateTimeShort = "2006-01-02 15:04"
	// LayoutDateSlash 斜杠日期（历史 CSV/表格导出里存在），如 2029/01/01
	LayoutDateSlash = "2006/01/02"
)

// flexibleTimeLayouts 是「面板 / CSV / 前端」可能传来的时间形态，顺序即优先级。
// 只在解析外部输入时使用；解析数据库存储值请用 ParseStoredTime。
var flexibleTimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",
	LayoutDateTime,
	LayoutDateTimeShort,
	LayoutDateSlash,
	LayoutDate,
}

// TimeFromUnixAuto 自动分辨秒与毫秒时间戳（1e12 以上按毫秒）。
// 前端 Date.getTime() 给的是毫秒，后端历史上又有秒级时间戳，两者必须都能接。
func TimeFromUnixAuto(ts int64) time.Time {
	if ts >= 1_000_000_000_000 {
		return time.UnixMilli(ts)
	}
	return time.Unix(ts, 0)
}

// ParseFlexibleTimeString 解析外部传入的时间字符串。
// 无时区信息时按本地时区解释（面板上填的「2029-01-01」就是本地零点）。
func ParseFlexibleTimeString(s string) (time.Time, error) {
	text := strings.TrimSpace(s)
	if text == "" {
		return time.Time{}, fmt.Errorf("时间不能为空")
	}
	// 纯数字字符串 = 时间戳（前端 Date.getTime() 给的是毫秒）
	if ts, err := strconv.ParseInt(text, 10, 64); err == nil {
		return TimeFromUnixAuto(ts), nil
	}
	for _, layout := range flexibleTimeLayouts {
		if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("时间格式不正确：%q（支持 2029-01-01、2029-01-01 12:00:00、2029-01-01T12:00:00+08:00 或时间戳）", text)
}

// ParseFlexibleTime 解析 JSON 反序列化后的任意时间值（string / 数字 / time.Time）。
// 解析失败必须由调用方明确报错：静默跳过会让接口回答「成功」而数据库里还是旧值。
func ParseFlexibleTime(v interface{}) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case *time.Time:
		if t == nil {
			return time.Time{}, fmt.Errorf("时间不能为空")
		}
		return *t, nil
	case string:
		return ParseFlexibleTimeString(t)
	case float64:
		return TimeFromUnixAuto(int64(t)), nil
	case int64:
		return TimeFromUnixAuto(t), nil
	case int:
		return TimeFromUnixAuto(int64(t)), nil
	default:
		return time.Time{}, fmt.Errorf("时间格式不正确")
	}
}

// ParseStoredTime 解析数据库/SQLite 里存下来的时间字符串。
//
// 与 ParseFlexibleTimeString 的区别：这里接受带纳秒与时区后缀的历史存储形态，
// 且**不**接受时间戳（存储列是文本），解析失败即为数据异常，必须报错。
func ParseStoredTime(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		LayoutDateTime,
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间字符串: %q", s)
}
