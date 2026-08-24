package utils

import (
	"strconv"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
)

// ---------- in-memory settings cache (5 min TTL) ----------
// 系统配置缓存，减少数据库查询，提升性能

var (
	cacheMu       sync.RWMutex
	settingsCache map[string]string
	lastCacheTime time.Time
	cacheTTL      = 5 * time.Minute // 从 30 秒增加到 5 分钟，减少数据库查询
)

// refreshCacheIfStale reloads all settings from DB when the cache is older than cacheTTL.
// Caller must NOT hold cacheMu.
func refreshCacheIfStale() {
	cacheMu.RLock()
	fresh := settingsCache != nil && time.Since(lastCacheTime) < cacheTTL
	cacheMu.RUnlock()
	if fresh {
		return
	}

	// Reload all settings in one query
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Find(&configs)

	m := make(map[string]string, len(configs))
	for _, c := range configs {
		m[c.Key] = c.Value
	}

	cacheMu.Lock()
	settingsCache = m
	lastCacheTime = time.Now()
	cacheMu.Unlock()
}

// InvalidateSettingsCache forces the next GetSetting/GetSettings call to reload from DB.
func InvalidateSettingsCache() {
	cacheMu.Lock()
	settingsCache = nil
	lastCacheTime = time.Time{}
	cacheMu.Unlock()
	// 配置变更同时失效公共配置缓存（站点名/logo/客户端下载链接等来自 config）
	InvalidatePublicCache("public_config")
}

// GetSettings reads multiple keys from the cached settings map.
func GetSettings(keys ...string) map[string]string {
	refreshCacheIfStale()
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	m := make(map[string]string, len(keys))
	for _, k := range keys {
		if v, ok := settingsCache[k]; ok {
			m[k] = v
		}
	}
	return m
}

// GetSetting reads a single key from the cached settings map.
func GetSetting(key string) string {
	refreshCacheIfStale()
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return settingsCache[key]
}

// IsBoolSetting checks if a setting is truthy ("true" or "1")
func IsBoolSetting(key string) bool {
	v := GetSetting(key)
	return v == "true" || v == "1"
}

// IsBoolSettingDefault checks if a setting is truthy, with a default if not set
func IsBoolSettingDefault(key string, defaultVal bool) bool {
	v := GetSetting(key)
	if v == "" {
		return defaultVal
	}
	return v == "true" || v == "1"
}

// GetIntSetting reads an integer setting with a default fallback
func GetIntSetting(key string, defaultVal int) int {
	v := GetSetting(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

// GetFloatSetting reads a float64 setting with a default fallback
func GetFloatSetting(key string, defaultVal float64) float64 {
	v := GetSetting(key)
	if v == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return defaultVal
	}
	return f
}

// ---------- 公共端点数据缓存（60s TTL，内存） ----------
// 供 /config /packages /announcements /payment/methods 等高访问公共端点复用，
// 避免每次请求都查库放大 DB 负载。

var (
	publicCacheMu   sync.RWMutex
	publicCacheData = make(map[string]publicCacheEntry)
)

type publicCacheEntry struct {
	data      interface{}
	expireAt  time.Time
}

// GetPublicCache 读取公共数据缓存；未命中或过期返回 nil
func GetPublicCache(key string) interface{} {
	publicCacheMu.RLock()
	defer publicCacheMu.RUnlock()
	entry, ok := publicCacheData[key]
	if !ok || time.Now().After(entry.expireAt) {
		return nil
	}
	return entry.data
}

// SetPublicCache 写入公共数据缓存（TTL 60s）
func SetPublicCache(key string, data interface{}) {
	publicCacheMu.Lock()
	defer publicCacheMu.Unlock()
	publicCacheData[key] = publicCacheEntry{data: data, expireAt: time.Now().Add(60 * time.Second)}
}

// InvalidatePublicCache 清除指定公共缓存（配置/套餐/公告变更时调用）
func InvalidatePublicCache(key string) {
	publicCacheMu.Lock()
	delete(publicCacheData, key)
	publicCacheMu.Unlock()
}
