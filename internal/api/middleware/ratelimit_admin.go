package middleware

import (
	"context"
	"strings"
	"time"

	"cboard/v2/internal/database"
)

// 限流器的可管理接口（后台「登录限制 / 解封」用）。
//
// 为什么需要：限流是按「IP + 接口」计数的，客户自己看不到计数、也没法清。
// 触发后只能干等窗口过去（登录 1 分钟，重置订阅 30 分钟），客服也没法处理。
// 这里把计数暴露出来：能列出当前被限的 IP、能按 IP 立即清零。
// Redis 侧清 key，纯内存侧清计数（两种模式都要覆盖，否则「解封了却还限着」）。

// RateLimitEntry 一条当前生效的限流记录。
type RateLimitEntry struct {
	Scope      string `json:"scope"`       // redis / memory
	Path       string `json:"path"`        // 接口路径，如 /api/v1/auth/login
	IP         string `json:"ip"`          // 来源 IP
	Count      int64  `json:"count"`       // 当前窗口内的请求数
	Limit      int    `json:"limit"`       // 该接口的每分钟上限
	TTLSeconds int64  `json:"ttl_seconds"` // 距离窗口结束的秒数
}

// limiterRegistry 记录创建过的内存限流器，供后台按 IP 清零。
var limiterRegistryMu = struct {
	mu   chanMutex
	list []*memRateLimiter
}{mu: make(chanMutex, 1)}

// chanMutex 用带缓冲 channel 实现的轻量互斥锁，避免为一个登记表引入新的依赖。
type chanMutex chan struct{}

func (m chanMutex) Lock()   { m <- struct{}{} }
func (m chanMutex) Unlock() { <-m }

func registerLimiter(rl *memRateLimiter) {
	limiterRegistryMu.mu.Lock()
	limiterRegistryMu.list = append(limiterRegistryMu.list, rl)
	limiterRegistryMu.mu.Unlock()
}

// ResetMemoryLimiters 清零指定 IP 在所有内存限流器中的计数。
// ip 为空表示清零全部。返回被清理的计数条目数。
func ResetMemoryLimiters(ip string) int {
	limiterRegistryMu.mu.Lock()
	list := make([]*memRateLimiter, len(limiterRegistryMu.list))
	copy(list, limiterRegistryMu.list)
	limiterRegistryMu.mu.Unlock()

	cleared := 0
	for _, rl := range list {
		rl.mu.Lock()
		for key := range rl.visitors {
			if ip == "" || key == ip {
				delete(rl.visitors, key)
				cleared++
			}
		}
		rl.mu.Unlock()
	}
	return cleared
}

// ListMemoryLimiters 列出内存限流器里当前处于「达到上限」状态的记录。
func ListMemoryLimiters() []RateLimitEntry {
	limiterRegistryMu.mu.Lock()
	list := make([]*memRateLimiter, len(limiterRegistryMu.list))
	copy(list, limiterRegistryMu.list)
	limiterRegistryMu.mu.Unlock()

	now := time.Now()
	entries := []RateLimitEntry{}
	for _, rl := range list {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if v.count >= rl.rate && now.Sub(v.lastSeen) <= rl.window {
				entries = append(entries, RateLimitEntry{
					Scope:      "memory",
					IP:         ip,
					Count:      int64(v.count),
					Limit:      rl.rate,
					TTLSeconds: int64(rl.window.Seconds() - now.Sub(v.lastSeen).Seconds()),
				})
			}
		}
		rl.mu.Unlock()
	}
	return entries
}

// ListRedisRateLimits 列出 Redis 中当前存在的限流计数（形如 ratelimit:<path>:<ip>）。
func ListRedisRateLimits() []RateLimitEntry {
	r := database.GetRedis()
	if r == nil {
		return []RateLimitEntry{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	entries := []RateLimitEntry{}
	var cursor uint64
	for {
		keys, next, err := r.Scan(ctx, cursor, "ratelimit:*", 200).Result()
		if err != nil {
			return entries
		}
		for _, key := range keys {
			path, ip := splitRateLimitKey(key)
			cnt, _ := r.Get(ctx, key).Int64()
			ttl, _ := r.TTL(ctx, key).Result()
			ttlSec := int64(0)
			if ttl > 0 {
				ttlSec = int64(ttl.Seconds())
			}
			entries = append(entries, RateLimitEntry{
				Scope:      "redis",
				Path:       path,
				IP:         ip,
				Count:      cnt,
				TTLSeconds: ttlSec,
			})
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return entries
}

// ClearRedisRateLimits 删除指定 IP 的限流 key（ip 为空表示全部），返回删除数量。
func ClearRedisRateLimits(ip string) int {
	r := database.GetRedis()
	if r == nil {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pattern := "ratelimit:*"
	if ip != "" {
		pattern = "ratelimit:*:" + ip
	}
	deleted := 0
	var cursor uint64
	for {
		keys, next, err := r.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			return deleted
		}
		if len(keys) > 0 {
			if n, err := r.Del(ctx, keys...).Result(); err == nil {
				deleted += int(n)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return deleted
}

// splitRateLimitKey 把 ratelimit:<path>:<ip> 拆成 path 与 ip。
//
// 不能简单按最后一个冒号切：IPv6 地址本身就带冒号
// （ratelimit:/api/v1/client/subscribe:240e:33d::1），
// 按最后一个冒号切会把 IP 切成 "1"、路径切出一截 IPv6。
// 路由路径都以 "/" 开头且不含冒号，因此从「最后一个 / 之后的第一个冒号」处切。
func splitRateLimitKey(key string) (path, ip string) {
	body := strings.TrimPrefix(key, "ratelimit:")
	slash := strings.LastIndex(body, "/")
	if slash < 0 {
		return body, ""
	}
	rest := body[slash:]
	idx := strings.Index(rest, ":")
	if idx < 0 {
		return body, ""
	}
	return body[:slash+idx], rest[idx+1:]
}
