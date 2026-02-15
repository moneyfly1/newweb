package middleware

import (
	"sync"
	"time"

	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int
	window   time.Duration
}

type visitor struct {
	count    int
	lastSeen time.Time
}

func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{visitors: make(map[string]*visitor), rate: rate, window: window}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, exists := rl.visitors[key]
	if !exists || time.Since(v.lastSeen) > rl.window {
		rl.visitors[key] = &visitor{count: 1, lastSeen: time.Now()}
		return true
	}
	if v.count >= rl.rate {
		return false
	}
	v.count++
	v.lastSeen = time.Now()
	return true
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(rl.window)
		rl.mu.Lock()
		for key, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.window {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit 频率限制中间件
func RateLimit(rate int, window time.Duration) gin.HandlerFunc {
	limiter := newRateLimiter(rate, window)
	return func(c *gin.Context) {
		if !limiter.allow(utils.GetRealClientIP(c)) {
			utils.TooManyRequests(c, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}
