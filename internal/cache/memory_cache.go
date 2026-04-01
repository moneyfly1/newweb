package cache

import (
	"sync"
	"time"
)

// TTLItem represents a cached item with an expiration time
type TTLItem struct {
	Value      interface{}
	ExpireTime time.Time
}

// MemoryCache represents an in-memory TTL cache
type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]TTLItem
	maxSize  int
}

const defaultMaxSize = 1000

var (
	memoryCache *MemoryCache
	memOnce     sync.Once
)

// GetMemoryCache gets the global memory cache instance
func GetMemoryCache() *MemoryCache {
	memOnce.Do(func() {
		memoryCache = &MemoryCache{
			items:   make(map[string]TTLItem),
			maxSize: defaultMaxSize,
		}
		go memoryCache.cleanup()
	})
	return memoryCache
}

// Set adds a value to the cache with the given time-to-live
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxSize {
		c.evictExpired()
	}

	c.items[key] = TTLItem{
		Value:      value,
		ExpireTime: time.Now().Add(ttl),
	}
}

func (c *MemoryCache) evictExpired() {
	now := time.Now()
	for k, v := range c.items {
		if now.After(v.ExpireTime) {
			delete(c.items, k)
		}
	}
}

func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		c.evictExpired()
		c.mu.Unlock()
	}
}

// Get gets a value from the cache. Returns nil and false if not found or expired.
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}
	if time.Now().After(item.ExpireTime) {
		// Do not delete here to avoid upgrading the lock, 
		// it will naturally be overwritten or simply considered missing.
		return nil, false
	}
	return item.Value, true
}
