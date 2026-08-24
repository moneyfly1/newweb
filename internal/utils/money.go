package utils

import "math"

// Round2 将金额保留两位小数（按分位舍入），统一浮点金额处理
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}
