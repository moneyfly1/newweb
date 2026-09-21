package utils

import (
	"testing"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupSettingsTestDB 为配置读取相关用例准备一个内存库
func setupSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	old := database.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db
	InvalidateSettingsCache()
	t.Cleanup(func() {
		database.DB = old
		InvalidateSettingsCache()
	})
	return db
}
