package handlers

import (
	"time"

	"cboard/v2/internal/utils"
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
//
// 解析本身已收敛到 utils.ParseFlexibleTime（与优惠券时间、CSV 导入时间共用同一实现）：
// 此前几处解析规则各不相同——同样的输入在用户管理能用、在订阅管理直接报错；
// 日期回退布局一个按 UTC 解释、一个按本地时区解释，写进库里的到期时刻能差 8 小时。
func parseExpireTimeParam(v interface{}) (time.Time, error) {
	return utils.ParseFlexibleTime(v)
}
