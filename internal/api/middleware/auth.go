package middleware

import (
	"strings"

	"cboard/v2/internal/config"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 声明
type Claims struct {
	UserID uint   `json:"user_id"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
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
		if models.IsTokenBlacklisted(database.GetDB(), tokenHash) {
			utils.Unauthorized(c, "Token 已失效")
			c.Abort()
			return
		}
		var user models.User
		if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
			utils.Unauthorized(c, "用户不存在")
			c.Abort()
			return
		}
		if !user.IsActive {
			utils.Forbidden(c, "账户已被禁用")
			c.Abort()
			return
		}
		c.Set("user", &user)
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
		if models.IsTokenBlacklisted(database.GetDB(), tokenHash) {
			c.Next()
			return
		}
		var user models.User
		if err := database.GetDB().First(&user, claims.UserID).Error; err == nil && user.IsActive {
			c.Set("user", &user)
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
	return c.Query("token")
}

// ParseToken 解析 JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
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
