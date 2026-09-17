package utils

import (
	"math"
	"strconv"
)

// Round2 将金额保留两位小数（按分位舍入），统一浮点金额处理
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// CentsToYuan 后台配置里的「分」→ 入账用的「元」。
//
// 为什么必须集中：签到奖励设置 checkin_min_reward / checkin_max_reward 存的是分，
// 管理面板按分展示成 ¥0.1 ~ ¥0.5，而入账时代码漏了这一步换算，
// 直接把「分」当「元」加到余额上——每个用户每天签到实际能拿到 ¥10 ~ ¥50
// （线上 balance_logs 里有 15 元的签到入账记录）。凡是读这两个配置项的地方，
// 入账与展示都必须经过同一个换算函数，不允许再手写 /100 或漏写。
func CentsToYuan(cents int) float64 {
	return Round2(float64(cents) / 100)
}

// PercentOf 按百分比计算金额：amount 的 percent%（如 10 表示 10%），结果保留两位小数。
//
// 为什么要有这个函数：优惠券折扣、代理佣金此前各写各的
// （`math.Round(a*v)/100`、`Round2(a*r/100)`），
// 同样的比例在不同入口能算出差 1 分钱的结果，用户看到的「应付金额」和
// 后台对账金额就对不上。四舍五入到分是唯一口径。
func PercentOf(amount, percent float64) float64 {
	return Round2(amount * percent / 100)
}

// YuanToCents 元 → 分（四舍五入）。
//
// 为什么必须舍入：支付创建与支付回调校验此前各写一套，一个用 int64(x*100)
// 截断、一个用 math.Round(x*100)，同一个订单两边算出的分数可能差 1 分；
// 统一成四舍五入后，创建与回调校验必然一致。
func YuanToCents(yuan float64) int64 {
	if math.IsNaN(yuan) || math.IsInf(yuan, 0) {
		return 0
	}
	return int64(math.Round(yuan * 100))
}

// YuanToCentsAtLeast 元 → 分，并保证不低于 minCents。
// 支付网关对最低金额有要求时，创建与校验必须用同一套下限规则。
func YuanToCentsAtLeast(yuan float64, minCents int64) int64 {
	cents := YuanToCents(yuan)
	if cents < minCents {
		return minCents
	}
	return cents
}

// ExchangeRate 读取后台配置的汇率，未配置/非法/非正数时回退默认值。
//
// 此前「默认值 + 读 setting + 解析 + >0 校验」这段逻辑在支付创建、
// 加密货币支付、支付回调校验里各抄了一遍，默认值与校验条件必须永远一致，
// 否则会出现「按 7.2 下单、按 7.0 校验」的金额不匹配。
func ExchangeRate(settingKey string, fallback float64) float64 {
	if settingKey == "" {
		return fallback
	}
	raw := GetSetting(settingKey)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
