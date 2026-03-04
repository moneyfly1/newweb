package utils

// IsBoolEnabled 检查字符串值是否表示启用
// 支持: "true", "1", "yes", "on"
func IsBoolEnabled(value string) bool {
	return value == "true" || value == "1" || value == "yes" || value == "on"
}

// IsBoolDisabled 检查字符串值是否表示禁用
// 支持: "false", "0", "no", "off", ""
func IsBoolDisabled(value string) bool {
	return value == "false" || value == "0" || value == "no" || value == "off" || value == ""
}

// DefaultString 返回字符串或默认值
func DefaultString(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// DefaultInt 返回整数或默认值
func DefaultInt(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}

// DefaultFloat 返回浮点数或默认值
func DefaultFloat(value, defaultValue float64) float64 {
	if value == 0 {
		return defaultValue
	}
	return value
}

// InStringSlice 检查字符串是否在切片中
func InStringSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// InUintSlice 检查 uint 是否在切片中
func InUintSlice(num uint, slice []uint) bool {
	for _, n := range slice {
		if n == num {
			return true
		}
	}
	return false
}

// UniqueStrings 去重字符串切片
func UniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, str := range slice {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}

// UniqueUints 去重 uint 切片
func UniqueUints(slice []uint) []uint {
	seen := make(map[uint]bool)
	result := make([]uint, 0, len(slice))
	for _, num := range slice {
		if !seen[num] {
			seen[num] = true
			result = append(result, num)
		}
	}
	return result
}

// TruncateString 截断字符串到指定长度
func TruncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	if maxLen <= 3 {
		return str[:maxLen]
	}
	return str[:maxLen-3] + "..."
}
