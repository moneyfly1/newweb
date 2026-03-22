package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"sync"
	"time"

	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

type csrfToken struct {
	token     string
	createdAt time.Time
}

type csrfStore struct {
	mu     sync.RWMutex
	tokens map[uint]*csrfToken // userID -> token
}

var store = &csrfStore{
	tokens: make(map[uint]*csrfToken),
}

const csrfTokenExpiry = 1 * time.Hour // 缩短到 1 小时

// 定期清理过期 token
func init() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			store.mu.Lock()
			now := time.Now()
			for userID, token := range store.tokens {
				if now.Sub(token.createdAt) > csrfTokenExpiry {
					delete(store.tokens, userID)
				}
			}
			store.mu.Unlock()
		}
	}()
}

// CSRFProtection CSRF 保护中间件（用于敏感操作）
func CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 仅对 POST/PUT/DELETE/PATCH 请求进行 CSRF 检查
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			utils.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}

		uid := userID.(uint)
		csrfToken := c.GetHeader("X-CSRF-Token")
		if csrfToken == "" {
			csrfToken = c.PostForm("csrf_token")
		}

		if !validateCSRFToken(uid, csrfToken) {
			utils.Forbidden(c, "CSRF token 无效或已过期")
			c.Abort()
			return
		}

		// 验证成功后立即轮换 token，防止重放
		rotateCSRFToken(uid)

		c.Next()
	}
}

// GetCSRFToken 获取或生成 CSRF token
func GetCSRFToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "请先登录")
		return
	}

	uid := userID.(uint)
	token := generateCSRFToken(uid)

	utils.Success(c, gin.H{"csrf_token": token})
}

func generateCSRFToken(userID uint) string {
	store.mu.Lock()
	defer store.mu.Unlock()

	// 检查是否已有有效 token
	if existing, ok := store.tokens[userID]; ok {
		if time.Since(existing.createdAt) < csrfTokenExpiry {
			return existing.token
		}
	}

	return generateNewToken(userID)
}

// rotateCSRFToken 在验证成功后强制生成新 token
func rotateCSRFToken(userID uint) {
	store.mu.Lock()
	defer store.mu.Unlock()
	generateNewToken(userID)
}

// generateNewToken 生成新 token 并存储（调用方需持有锁）
func generateNewToken(userID uint) string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	token := base64.URLEncoding.EncodeToString(b)

	store.tokens[userID] = &csrfToken{
		token:     token,
		createdAt: time.Now(),
	}

	return token
}

func validateCSRFToken(userID uint, token string) bool {
	if token == "" {
		return false
	}

	store.mu.RLock()
	defer store.mu.RUnlock()

	stored, ok := store.tokens[userID]
	if !ok {
		return false
	}

	// 检查是否过���
	if time.Since(stored.createdAt) > csrfTokenExpiry {
		return false
	}

	// 常量时间比较防止时序攻击
	return subtle.ConstantTimeCompare([]byte(token), []byte(stored.token)) == 1
}
