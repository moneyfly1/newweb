package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSubscriptionCompatTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := database.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Subscription{}, &models.SubscriptionReset{}, &models.SystemConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db
	t.Cleanup(func() {
		database.DB = oldDB
	})
	return db
}

func TestFindSubscriptionByAccessTokenAcceptsResetHistoryToken(t *testing.T) {
	db := setupSubscriptionCompatTestDB(t)

	sub := models.Subscription{
		UserID:          1,
		SubscriptionURL: "new-token",
		DeviceLimit:     3,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	oldToken := "old-token"
	newToken := "new-token"
	if err := db.Create(&models.SubscriptionReset{
		UserID:             sub.UserID,
		SubscriptionID:     sub.ID,
		ResetType:          "manual",
		OldSubscriptionURL: &oldToken,
		NewSubscriptionURL: &newToken,
	}).Error; err != nil {
		t.Fatalf("create reset: %v", err)
	}

	got, err := findSubscriptionByAccessToken(db, oldToken)
	if err != nil {
		t.Fatalf("find by old token: %v", err)
	}
	if got.ID != sub.ID || got.SubscriptionURL != newToken {
		t.Fatalf("old token should resolve to current subscription, got id=%d token=%q", got.ID, got.SubscriptionURL)
	}
}

func TestLegacySubscribeRouteUsesCurrentSubscriptionControls(t *testing.T) {
	db := setupSubscriptionCompatTestDB(t)

	if err := db.Create(&models.SystemConfig{Key: "site_url", Value: "https://example.com", IsPublic: true}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}
	sub := models.Subscription{
		UserID:          1,
		SubscriptionURL: "new-token",
		DeviceLimit:     3,
		IsActive:        false,
		Status:          "disabled",
		ExpireTime:      time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	oldToken := "old-token"
	newToken := "new-token"
	if err := db.Create(&models.SubscriptionReset{
		UserID:             sub.UserID,
		SubscriptionID:     sub.ID,
		ResetType:          "admin_reset",
		OldSubscriptionURL: &oldToken,
		NewSubscriptionURL: &newToken,
	}).Error; err != nil {
		t.Fatalf("create reset: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/subscribe/:url", GetSubscription)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscribe/old-token?type=clash", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "订阅已失效") {
		t.Fatalf("legacy token should be controlled by current disabled subscription, body=%s", w.Body.String())
	}
}

func TestLegacySubWildcardRouteParsesFormatAndToken(t *testing.T) {
	db := setupSubscriptionCompatTestDB(t)

	sub := models.Subscription{
		UserID:          1,
		SubscriptionURL: "new-token",
		DeviceLimit:     3,
		IsActive:        false,
		Status:          "disabled",
		ExpireTime:      time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	oldToken := "old-token"
	newToken := "new-token"
	if err := db.Create(&models.SubscriptionReset{
		UserID:             sub.UserID,
		SubscriptionID:     sub.ID,
		ResetType:          "admin_reset",
		OldSubscriptionURL: &oldToken,
		NewSubscriptionURL: &newToken,
	}).Error; err != nil {
		t.Fatalf("create reset: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/sub/*path", GetSubscription)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sub/clash/old-token", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); !strings.Contains(got, "text/yaml") {
		t.Fatalf("legacy clash route content-type = %q, want text/yaml", got)
	}
	if !strings.Contains(w.Body.String(), "订阅已失效") {
		t.Fatalf("legacy wildcard token should be controlled by current disabled subscription, body=%s", w.Body.String())
	}
}
