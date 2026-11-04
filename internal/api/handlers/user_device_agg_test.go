package handlers

import (
	"testing"
	"time"

	"cboard/v2/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 用户列表「设备 / 客户端」聚合的回归测试：
// 同一客户多台设备、多个客户端（含 Unknown/空值）时，聚合结果要正确 ——
// 设备数、在线数（15 分钟窗口）、客户端去重且不把 Unknown 当成客户端名。
func TestLoadUserDeviceAggregates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Device{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	uid := int64(7)
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	strPtr := func(s string) *string { return &s }
	timePtr := func(t time.Time) *time.Time { return &t }

	devices := []models.Device{
		{UserID: &uid, SubscriptionID: 1, SoftwareName: strPtr("Clash Verge"), DeviceName: strPtr("PC"), IsActive: true, LastAccess: now, LastHeartbeat: timePtr(now)},
		{UserID: &uid, SubscriptionID: 1, SoftwareName: strPtr("Shadowrocket"), DeviceName: strPtr("iPhone"), IsActive: true, LastAccess: old},
		{UserID: &uid, SubscriptionID: 1, SoftwareName: strPtr("Unknown"), DeviceName: strPtr("未知设备"), IsActive: true, LastAccess: old.Add(-time.Minute)},
		{UserID: &uid, SubscriptionID: 1, SoftwareName: nil, DeviceName: strPtr("空客户端"), IsActive: true, LastAccess: old.Add(-2 * time.Minute)},
		{UserID: &uid, SubscriptionID: 1, SoftwareName: strPtr("已删除设备"), IsActive: false, LastAccess: now},
	}
	for i := range devices {
		if err := db.Create(&devices[i]).Error; err != nil {
			t.Fatalf("写入设备失败: %v", err)
		}
	}
	// GORM 对 bool 零值 + `default:true` 会写默认值，删除态设备需显式更新
	if err := db.Model(&models.Device{}).Where("id = ?", devices[4].ID).Update("is_active", false).Error; err != nil {
		t.Fatalf("更新设备状态失败: %v", err)
	}

	agg := loadUserDeviceAggregates(db, []uint{7, 8})
	got := agg[7]
	if got == nil {
		t.Fatal("用户 7 应返回设备聚合")
	}
	if got.count != 4 {
		t.Fatalf("活跃设备数应为 4（不含 is_active=false），实际 %d", got.count)
	}
	if got.online != 1 {
		t.Fatalf("在线数应为 1（只有 15 分钟内有心跳/拉取的那台），实际 %d", got.online)
	}
	if len(got.clients) != 2 || got.clients[0] != "Clash Verge" || got.clients[1] != "Shadowrocket" {
		t.Fatalf("客户端去重/排除 Unknown 失败: %v", got.clients)
	}
	if got.lastActive == nil {
		t.Fatal("最后活跃时间不应为空")
	}
	if agg[8] != nil {
		t.Fatal("没有设备的用户不应产生聚合条目")
	}
}
