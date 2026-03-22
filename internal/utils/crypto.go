package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// 加密随机数生成失败是严重安全事件，不使用可预测的降级方案
			panic(fmt.Sprintf("crypto/rand 失败，无法安全生成随机字符串: %v", err))
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func GenerateVerificationCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		panic(fmt.Sprintf("crypto/rand 失败，无法安全生成验证码: %v", err))
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func SHA256Hash(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}
