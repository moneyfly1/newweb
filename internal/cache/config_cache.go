package cache

import (
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
)

// ConfigCache 系统配置缓存
type ConfigCache struct {
	mu      sync.RWMutex
	configs map[string]string
	lastUpdate time.Time
	ttl     time.Duration
}

var (
	configCache *ConfigCache
	once        sync.Once
)

// GetConfigCache 获取配置缓存实例（单例）
func GetConfigCache() *ConfigCache {
	once.Do(func() {
		configCache = &ConfigCache{
			configs: make(map[string]string),
			ttl:     5 * time.Minute, // 5分钟过期
		}
	})
	return configCache
}

// Get 获取单个配置
func (c *ConfigCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 检查是否过期
	if time.Since(c.lastUpdate) > c.ttl {
		return "", false
	}

	value, ok := c.configs[key]
	return value, ok
}

// GetMultiple 获取多个配置
func (c *ConfigCache) GetMultiple(keys []string) map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 检查是否过期
	if time.Since(c.lastUpdate) > c.ttl {
		return nil
	}

	result := make(map[string]string)
	for _, key := range keys {
		if value, ok := c.configs[key]; ok {
			result[key] = value
		}
	}
	return result
}

// Set 设置单个配置
func (c *ConfigCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.configs[key] = value
	c.lastUpdate = time.Now()
}

// SetMultiple 批量设置配置
func (c *ConfigCache) SetMultiple(configs map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, value := range configs {
		c.configs[key] = value
	}
	c.lastUpdate = time.Now()
}

// Refresh 刷新缓存（从数据库重新加载）
func (c *ConfigCache) Refresh() error {
	db := database.GetDB()
	var configs []models.SystemConfig
	if err := db.Find(&configs).Error; err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.configs = make(map[string]string)
	for _, config := range configs {
		c.configs[config.Key] = config.Value
	}
	c.lastUpdate = time.Now()

	return nil
}

// Clear 清空缓存
func (c *ConfigCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.configs = make(map[string]string)
	c.lastUpdate = time.Time{}
}

// IsExpired 检查缓存是否过期
func (c *ConfigCache) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Since(c.lastUpdate) > c.ttl
}
