package utils

import (
	"strconv"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
)

// GetSettings reads multiple keys from system_configs and returns a map
func GetSettings(keys ...string) map[string]string {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("`key` IN ?", keys).Find(&configs)
	m := make(map[string]string, len(configs))
	for _, c := range configs {
		m[c.Key] = c.Value
	}
	return m
}

// GetSetting reads a single key from system_configs
func GetSetting(key string) string {
	db := database.GetDB()
	var cfg models.SystemConfig
	if err := db.Where("`key` = ?", key).First(&cfg).Error; err != nil {
		return ""
	}
	return cfg.Value
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
