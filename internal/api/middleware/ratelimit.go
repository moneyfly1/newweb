package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// memRateLimiter is a fallback in-memory rate limiter based on fixed window counter
type memRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int
	window   time.Duration
}

type visitor struct {
	count    int
	lastSeen time.Time
}

const maxVisitors = 10000

func newMemRateLimiter(rate int, window time.Duration) *memRateLimiter {
	rl := &memRateLimiter{visitors: make(map[string]*visitor), rate: rate, window: window}
	go rl.cleanup()
	return rl
}

func (rl *memRateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if len(rl.visitors) >= maxVisitors {
		now := time.Now()
		for k, v := range rl.visitors {
			if now.Sub(v.lastSeen) > rl.window {
				delete(rl.visitors, k)
			}
		}
		if len(rl.visitors) >= maxVisitors {
			return false
		}
	}

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

func (rl *memRateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, v := range rl.visitors {
			if now.Sub(v.lastSeen) > rl.window {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit 频率限制中间件 (优先使用 Redis，回退到内存)
func RateLimit(rate int, window time.Duration) gin.HandlerFunc {
	memFallback := newMemRateLimiter(rate, window)
	
	return func(c *gin.Context) {
		clientIP := utils.GetRealClientIP(c)
		r := database.GetRedis()
		
		allowed := true
		
		if r != nil {
			// Redis sliding window / fixed window logic
			ctx := context.Background()
			key := fmt.Sprintf("ratelimit:%s:%s", c.FullPath(), clientIP)
			
			// increment counter
			cnt, err := r.Incr(ctx, key).Result()
			if err == nil {
				if cnt == 1 {
					// First request, set expiration
					r.Expire(ctx, key, window)
				}
				if int(cnt) > rate {
					allowed = false
				}
			} else {
				// Redis error (e.g. timeout), fallback
				allowed = memFallback.allow(clientIP)
			}
		} else {
			allowed = memFallback.allow(clientIP)
		}

		if !allowed {
			utils.TooManyRequests(c, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}
