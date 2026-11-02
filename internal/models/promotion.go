package models

import "time"

// Promotion 促销活动：限时折扣 / 新用户 / 召回 / 会员日。
//
// 对齐 myweb 的 promotions 表（去掉了 JSONString 依赖，用普通 string 存描述）。
type Promotion struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"not null" json:"name"`
	Type          string    `gorm:"not null;index" json:"type"`           // flash_sale / new_user / recall / member_day
	DiscountType  string    `gorm:"not null" json:"discount_type"`        // percentage / fixed
	DiscountValue float64   `gorm:"not null" json:"discount_value"`       // percentage: 0.9 = 9折；fixed: 减免金额
	MinAmount     float64   `json:"min_amount"`                           // 订单最低金额
	MaxDiscount   float64   `json:"max_discount"`                         // 单笔最多减免
	StartTime     time.Time `gorm:"not null;index:idx_promotion_active,priority:2" json:"start_time"`
	EndTime       time.Time `gorm:"not null;index:idx_promotion_active,priority:3" json:"end_time"`
	IsActive      bool      `gorm:"index;index:idx_promotion_active,priority:1" json:"is_active"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Promotion) TableName() string { return "promotions" }

// PromotionParticipation 用户参与记录（新用户/召回类活动防重复领取）。
type PromotionParticipation struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	PromotionID    uint      `gorm:"index;not null" json:"promotion_id"`
	UserID         uint      `gorm:"uniqueIndex:idx_promo_user;index;not null" json:"user_id"`
	OrderID        *int64    `json:"order_id"`
	DiscountAmount float64   `json:"discount_amount"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (PromotionParticipation) TableName() string { return "promotion_participations" }
