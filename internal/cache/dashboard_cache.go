package cache

import (
	"context"
	"encoding/json"
	"time"

	"cboard/v2/internal/database"
)

// SetDashboardCache sets the dashboard data in cache
func SetDashboardCache(key string, data interface{}, ttl time.Duration) {
	// Try Redis first
	r := database.GetRedis()
	if r != nil {
		bytes, err := json.Marshal(data)
		if err == nil {
			r.Set(context.Background(), key, string(bytes), ttl)
			return
		}
	}

	// Fallback to Memory
	GetMemoryCache().Set(key, data, ttl)
}

// GetDashboardCache gets the dashboard data from cache
func GetDashboardCache(key string, dest interface{}) bool {
	// Try Redis first
	r := database.GetRedis()
	if r != nil {
		val, err := r.Get(context.Background(), key).Result()
		if err == nil && val != "" {
			if json.Unmarshal([]byte(val), dest) == nil {
				return true
			}
		}
	}

	// Fallback to memory
	if val, ok := GetMemoryCache().Get(key); ok {
		// Because it's an interface inside memory cache, we copy it via JSON to match Redis semantics
		bytes, err := json.Marshal(val)
		if err == nil {
			if json.Unmarshal(bytes, dest) == nil {
				return true
			}
		}
	}

	return false
}

// ClearDashboardCache aggressively clears the cached entry
func ClearDashboardCache(key string) {
	r := database.GetRedis()
	if r != nil {
		r.Del(context.Background(), key)
	}
	// Memory cache will expire by TTL naturally, or we can just let it expire. We don't have a Delete in MemoryCache right now.
}
