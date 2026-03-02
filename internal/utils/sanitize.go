package utils

import (
	"html"
	"regexp"
	"strings"
)

// SanitizeInput 清理用户输入，防止 XSS
func SanitizeInput(input string) string {
	// HTML 转义
	input = html.EscapeString(input)

	// 移除控制字符
	input = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, input)

	return strings.TrimSpace(input)
}

// SanitizeUsername 清理用户名
func SanitizeUsername(username string) string {
	username = strings.TrimSpace(username)

	// 只允许字母、数字、下划线、中文
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\p{Han}]`)
	username = reg.ReplaceAllString(username, "")

	return username
}

// SanitizeEmail 清理邮箱
func SanitizeEmail(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))

	// 基本邮箱格式验证
	reg := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	if !reg.MatchString(email) {
		return ""
	}

	return email
}

// ValidateNoSQLInjection 检查是否包含 SQL 注入特征
func ValidateNoSQLInjection(input string) bool {
	// 虽然 GORM 已参数化，但额外检查
	dangerous := []string{
		"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_",
		"exec", "execute", "select", "insert", "update", "delete",
		"drop", "create", "alter", "union", "script",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range dangerous {
		if strings.Contains(lowerInput, pattern) {
			return false
		}
	}

	return true
}

// ValidateNoXSS 检查是否包含 XSS 特征
func ValidateNoXSS(input string) bool {
	dangerous := []string{
		"<script", "</script", "javascript:", "onerror=", "onload=",
		"<iframe", "<object", "<embed", "eval(", "alert(",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range dangerous {
		if strings.Contains(lowerInput, pattern) {
			return false
		}
	}

	return true
}

// ValidateNoPathTraversal 检查是否包含路径遍历特征
func ValidateNoPathTraversal(input string) bool {
	dangerous := []string{
		"../", "..\\", "..", "%2e%2e", "%252e%252e",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range dangerous {
		if strings.Contains(lowerInput, pattern) {
			return false
		}
	}

	return true
}

// SanitizeFilename 清理文件名
func SanitizeFilename(filename string) string {
	// 移除路径分隔符
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")
	filename = strings.ReplaceAll(filename, "..", "")

	// 只保留安全字符
	reg := regexp.MustCompile(`[^a-zA-Z0-9._\-\p{Han}]`)
	filename = reg.ReplaceAllString(filename, "_")

	return filename
}

// ValidateURL 验证 URL 安全性
func ValidateURL(url string) bool {
	// 防止 SSRF 攻击
	dangerous := []string{
		"file://", "gopher://", "dict://", "ftp://",
		"localhost", "127.0.0.1", "0.0.0.0", "::1",
		"169.254.", "10.", "172.16.", "192.168.",
	}

	lowerURL := strings.ToLower(url)
	for _, pattern := range dangerous {
		if strings.Contains(lowerURL, pattern) {
			return false
		}
	}

	// 只允许 http/https
	if !strings.HasPrefix(lowerURL, "http://") && !strings.HasPrefix(lowerURL, "https://") {
		return false
	}

	return true
}
