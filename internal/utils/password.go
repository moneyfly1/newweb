package utils

import (
	"fmt"
	"unicode"
)

// ValidatePasswordStrength checks that a password meets minimum requirements:
// at least 8 characters, contains both letters and digits.
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("密码长度不能少于8位")
	}
	hasLetter := false
	hasDigit := false
	for _, ch := range password {
		if unicode.IsLetter(ch) {
			hasLetter = true
		}
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("密码必须包含字母和数字")
	}
	return nil
}
