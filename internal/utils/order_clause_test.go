package utils

import (
	"testing"
	"time"

	"cboard/v2/internal/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 用例背景（用户反馈「用户列表按注册时间排序不对」）：
// SQLite 里时间存的是字符串，而历史数据格式并不统一（带不带时区、分隔符是 T 还是空格）。
// 直接按文本 ORDER BY 会得到错误顺序，必须先用 datetime() 归一化。
// 这里用真实会踩到的两种数据形态证明修复有效。

type orderTestRow struct {
	ID        uint   `gorm:"primaryKey"`
	CreatedAt string `gorm:"column:created_at"`
}

func (orderTestRow) TableName() string { return "order_test_rows" }

func setupOrderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	old := database.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&orderTestRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db
	t.Cleanup(func() { database.DB = old })
	return db
}

// 形态一：同一天里，一条用 T 分隔、一条用空格分隔 —— 文本比较时 'T'(0x54) > ' '(0x20)，
// 空格那条永远排前面，即使它的时间更晚。
func TestOrderClauseSortsMixedSeparatorsChronologically(t *testing.T) {
	db := setupOrderTestDB(t)
	rows := []orderTestRow{
		{CreatedAt: "2026-09-16T08:00:00"}, // 08:00（T 分隔）
		{CreatedAt: "2026-09-16 09:00:00"}, // 09:00（空格分隔）
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	var got []orderTestRow
	p := Pagination{Sort: "created_at", Order: "asc"}
	if err := db.Order(p.OrderClause()).Find(&got).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 rows, got %d", len(got))
	}
	if got[0].CreatedAt != "2026-09-16T08:00:00" {
		t.Fatalf("升序第一条应为 08:00（T 分隔），实际 %s —— 说明时间列没有归一化就按文本排序了", got[0].CreatedAt)
	}

	var desc []orderTestRow
	pd := Pagination{Sort: "created_at", Order: "desc"}
	if err := db.Order(pd.OrderClause()).Find(&desc).Error; err != nil {
		t.Fatalf("query desc: %v", err)
	}
	if desc[0].CreatedAt != "2026-09-16 09:00:00" {
		t.Fatalf("降序第一条应为 09:00（最新的在前），实际 %s", desc[0].CreatedAt)
	}
}

// 形态二：格式相同但时区不同 —— 文本排序只看墙上时间，会把 +00:00 的 08:00 排到
// +08:00 的 09:00（真实时刻更早）前面。
func TestOrderClauseHonoursTimezoneOffsets(t *testing.T) {
	db := setupOrderTestDB(t)
	rows := []orderTestRow{
		{CreatedAt: "2026-09-16 09:00:00+08:00"}, // 真实时刻 01:00Z（更早）
		{CreatedAt: "2026-09-16 08:00:00+00:00"}, // 真实时刻 08:00Z（更晚）
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	var got []orderTestRow
	p := Pagination{Sort: "created_at", Order: "asc"}
	if err := db.Order(p.OrderClause()).Find(&got).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if got[0].CreatedAt != "2026-09-16 09:00:00+08:00" {
		t.Fatalf("升序第一条应为 +08:00 的 09:00（真实时刻 01:00Z），实际 %s", got[0].CreatedAt)
	}
}

// 非时间列不受影响，仍然是普通列排序
func TestOrderClauseKeepsPlainColumns(t *testing.T) {
	p := Pagination{Sort: "id", Order: "desc"}
	if tc := p.OrderClause(); tc != "id desc" {
		t.Fatalf("普通列排序被改坏了: %s", tc)
	}
}

// 时间列在 SQLite 下必须包 datetime()，且解析不了的值回退原文（不清成 NULL）
func TestOrderClauseWrapsDateColumnsOnSQLite(t *testing.T) {
	setupOrderTestDB(t)
	p := Pagination{Sort: "created_at", Order: "desc"}
	tc := p.OrderClause()
	if tc != "COALESCE(datetime(created_at), created_at) desc" {
		t.Fatalf("时间列排序语句不符合预期: %s", tc)
	}
}

var _ = time.Now
