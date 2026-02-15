package utils

import (
	"strconv"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
)

// ---------- in-memory settings cache (30 s TTL) ----------

var (
	cacheMu       sync.RWMutex
	settingsCache map[string]string
	lastCacheTime time.Time
	cacheTTL      = 30 * time.Second
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
