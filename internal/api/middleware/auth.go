package middleware

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"cboard/v2/internal/config"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type userCacheEntry struct {
	user     *models.User
	expireAt time.Time
}

var (
	userCache      = make(map[uint]userCacheEntry)
	userCacheMu    sync.RWMutex
	userCacheTTL   = 5 * time.Minute
	blacklistCache = make(map[string]time.Time)
	blacklistMu    sync.RWMutex
)

// Claims JWT 声明
type Claims struct {
	UserID uint   `json:"user_id"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

func init() {
	go cacheCleanup()
}

func cacheCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		userCacheMu.Lock()
		for k, v := range userCache {
			if now.After(v.expireAt) {
				delete(userCache, k)
			}
		}
		userCacheMu.Unlock()

		blacklistMu.Lock()
		for k, v := range blacklistCache {
			if now.After(v) {
				delete(blacklistCache, k)
			}
		}
		blacklistMu.Unlock()
	}
}

func isTokenBlacklisted(tokenHash string) bool {
	blacklistMu.RLock()
	expireAt, exists := blacklistCache[tokenHash]
	blacklistMu.RUnlock()
	if exists {
		return time.Now().Before(expireAt)
	}

	if models.IsTokenBlacklisted(database.GetDB(), tokenHash) {
		blacklistMu.Lock()
		blacklistCache[tokenHash] = time.Now().Add(24 * time.Hour)
		blacklistMu.Unlock()
		return true
	}
	return false
}

func getUserByID(userID uint) (*models.User, error) {
	now := time.Now()
	userCacheMu.RLock()
	if cached, ok := userCache[userID]; ok && now.Before(cached.expireAt) {
		userCacheMu.RUnlock()
		return cached.user, nil
	}
	userCacheMu.RUnlock()

	var user models.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		return nil, err
	}

	userCacheMu.Lock()
	userCache[userID] = userCacheEntry{
		user:     &user,
		expireAt: now.Add(userCacheTTL),
	}
	userCacheMu.Unlock()
	return &user, nil
}

// AuthRequired 要求用户已登录
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			utils.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}
		claims, err := ParseToken(tokenString)
		if err != nil {
			utils.Unauthorized(c, "Token 无效或已过期")
			c.Abort()
			return
		}
		tokenHash := utils.SHA256Hash(tokenString)
		if isTokenBlacklisted(tokenHash) {
			utils.Unauthorized(c, "Token 已失效")
			c.Abort()
			return
		}
		user, err := getUserByID(claims.UserID)
		if err != nil {
			utils.Unauthorized(c, "用户不存在")
			c.Abort()
			return
		}
		if !user.IsActive {
			utils.Forbidden(c, "账户已被禁用")
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("token", tokenString)
		c.Next()
	}
}

// AdminRequired 要求管理员权限
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			utils.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}
		if !user.(*models.User).IsAdmin {
			utils.Forbidden(c, "需要管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

// OptionalAuth 可选认证
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			c.Next()
			return
		}
		claims, err := ParseToken(tokenString)
		if err != nil {
			c.Next()
			return
		}
		tokenHash := utils.SHA256Hash(tokenString)
		if isTokenBlacklisted(tokenHash) {
			c.Next()
			return
		}
		user, err := getUserByID(claims.UserID)
		if err == nil && user.IsActive {
			c.Set("user", user)
			c.Set("user_id", user.ID)
		}
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// ParseToken 解析 JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.GetSecretKey()), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.Type != "access" {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}
