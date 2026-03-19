package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"time"

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
			// 避免因熵源异常导致服务崩溃，降级为确定性哈希派生并记录告警
			fallback := deriveFallbackBytes(length - i)
			for j := i; j < len(b); j++ {
				b[j] = charset[int(fallback[j-i])%len(charset)]
			}
			SysError("security", fmt.Sprintf("随机数生成失败，已使用降级方案: %v", err))
			return string(b)
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func GenerateVerificationCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		// 避免熵源异常造成流程中断
		fallback := deriveFallbackBytes(8)
		var v int
		for _, b := range fallback {
			v = (v << 5) ^ int(b)
		}
		if v < 0 {
			v = -v
		}
		SysError("security", fmt.Sprintf("验证码随机数生成失败，已使用降级方案: %v", err))
		return fmt.Sprintf("%06d", (v%900000)+100000)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func SHA256Hash(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func deriveFallbackBytes(length int) []byte {
	if length <= 0 {
		return nil
	}
	out := make([]byte, 0, length)
	seed := fmt.Sprintf("%d:%d:%d", time.Now().UnixNano(), os.Getpid(), length)
	for len(out) < length {
		hash := sha256.Sum256([]byte(seed))
		out = append(out, hash[:]...)
		seed = hex.EncodeToString(hash[:])
	}
	return out[:length]
}
